package resolver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/assets"
	"github.com/smartcontractkit/chainlink/v2/core/auth"
	"github.com/smartcontractkit/chainlink/v2/core/bridges"
	"github.com/smartcontractkit/chainlink/v2/core/logger/audit"
	"github.com/smartcontractkit/chainlink/v2/core/services/blockhashstore"
	"github.com/smartcontractkit/chainlink/v2/core/services/chainlink"
	"github.com/smartcontractkit/chainlink/v2/core/services/gateway"
	"github.com/smartcontractkit/chainlink/v2/core/services/job"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/csakey"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/p2pkey"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/vrfkey"
	"github.com/smartcontractkit/chainlink/v2/core/services/vrf/vrfcommon"
	"github.com/smartcontractkit/chainlink/v2/core/services/webhook"
	"github.com/smartcontractkit/chainlink/v2/core/store/models"
	"github.com/smartcontractkit/chainlink/v2/core/utils"
	"github.com/smartcontractkit/chainlink/v2/core/utils/stringutils"
	webauth "github.com/smartcontractkit/chainlink/v2/core/web/auth"
)

type Resolver struct {
	App chainlink.Application
}

type createBridgeInput struct {
	Name                   string
	URL                    string
	Confirmations          int32
	MinimumContractPayment string
}

// CreateBridge creates a new bridge.
func (r *Resolver) CreateBridge(ctx context.Context, args struct{ Input createBridgeInput }) (*CreateBridgePayloadResolver, error) {
	if err := authenticateUserCanEdit(ctx); err != nil {
		return nil, err
	}

	var webURL models.WebURL
	if len(args.Input.URL) != 0 {
		rURL, err := url.ParseRequestURI(args.Input.URL)
		if err != nil {
			return nil, err
		}
		webURL = models.WebURL(*rURL)
	}
	minContractPayment := &assets.Link{}
	if err := minContractPayment.UnmarshalText([]byte(args.Input.MinimumContractPayment)); err != nil {
		return nil, err
	}

	btr := &bridges.BridgeTypeRequest{
		Name:                   bridges.BridgeName(args.Input.Name),
		URL:                    webURL,
		Confirmations:          uint32(args.Input.Confirmations),
		MinimumContractPayment: minContractPayment,
	}

	bta, bt, err := bridges.NewBridgeType(btr)
	if err != nil {
		return nil, err
	}
	orm := r.App.BridgeORM()
	if err = ValidateBridgeType(btr); err != nil {
		return nil, err
	}
	if err = ValidateBridgeTypeUniqueness(btr, orm); err != nil {
		return nil, err
	}
	if err := orm.CreateBridgeType(bt); err != nil {
		return nil, err
	}

	r.App.GetAuditLogger().Audit(audit.BridgeCreated, map[string]interface{}{
		"bridgeName":                   bta.Name,
		"bridgeConfirmations":          bta.Confirmations,
		"bridgeMinimumContractPayment": bta.MinimumContractPayment,
		"bridgeURL":                    bta.URL,
	})

	return NewCreateBridgePayload(*bt, bta.IncomingToken), nil
}

func (r *Resolver) CreateCSAKey(ctx context.Context) (*CreateCSAKeyPayloadResolver, error) {
	if err := authenticateUserCanEdit(ctx); err != nil {
		return nil, err
	}

	key, err := r.App.GetKeyStore().CSA().Create()
	if err != nil {
		if errors.Is(err, keystore.ErrCSAKeyExists) {
			return NewCreateCSAKeyPayload(nil, err), nil
		}

		return nil, err
	}

	r.App.GetAuditLogger().Audit(audit.CSAKeyCreated, map[string]interface{}{
		"CSAPublicKey": key.PublicKey,
		"CSVersion":    key.Version,
	})

	return NewCreateCSAKeyPayload(&key, nil), nil
}

func (r *Resolver) DeleteCSAKey(ctx context.Context, args struct {
	ID graphql.ID
}) (*DeleteCSAKeyPayloadResolver, error) {
	if err := authenticateUserIsAdmin(ctx); err != nil {
		return nil, err
	}

	key, err := r.App.GetKeyStore().CSA().Delete(string(args.ID))
	if err != nil {
		if errors.As(err, &keystore.KeyNotFoundError{}) {
			return NewDeleteCSAKeyPayload(csakey.KeyV2{}, err), nil
		}

		return nil, err
	}

	r.App.GetAuditLogger().Audit(audit.CSAKeyDeleted, map[string]interface{}{"id": args.ID})

	return NewDeleteCSAKeyPayload(key, nil), nil
}

type updateBridgeInput struct {
	Name                   string
	URL                    string
	Confirmations          int32
	MinimumContractPayment string
}

func (r *Resolver) UpdateBridge(ctx context.Context, args struct {
	ID    graphql.ID
	Input updateBridgeInput
}) (*UpdateBridgePayloadResolver, error) {
	if err := authenticateUserCanEdit(ctx); err != nil {
		return nil, err
	}

	var webURL models.WebURL
	if len(args.Input.URL) != 0 {
		rURL, err := url.ParseRequestURI(args.Input.URL)
		if err != nil {
			return nil, err
		}
		webURL = models.WebURL(*rURL)
	}
	minContractPayment := &assets.Link{}
	if err := minContractPayment.UnmarshalText([]byte(args.Input.MinimumContractPayment)); err != nil {
		return nil, err
	}

	btr := &bridges.BridgeTypeRequest{
		Name:                   bridges.BridgeName(args.Input.Name),
		URL:                    webURL,
		Confirmations:          uint32(args.Input.Confirmations),
		MinimumContractPayment: minContractPayment,
	}

	taskType, err := bridges.ParseBridgeName(string(args.ID))
	if err != nil {
		return nil, err
	}

	// Find the bridge
	orm := r.App.BridgeORM()
	bridge, err := orm.FindBridge(taskType)
	if errors.Is(err, sql.ErrNoRows) {
		return NewUpdateBridgePayload(nil, err), nil
	}
	if err != nil {
		return nil, err
	}

	// Update the bridge
	if err := ValidateBridgeType(btr); err != nil {
		return nil, err
	}

	if err := orm.UpdateBridgeType(&bridge, btr); err != nil {
		return nil, err
	}

	r.App.GetAuditLogger().Audit(audit.BridgeUpdated, map[string]interface{}{
		"bridgeName":                   bridge.Name,
		"bridgeConfirmations":          bridge.Confirmations,
		"bridgeMinimumContractPayment": bridge.MinimumContractPayment,
		"bridgeURL":                    bridge.URL,
	})

	return NewUpdateBridgePayload(&bridge, nil), nil
}

func (r *Resolver) DeleteBridge(ctx context.Context, args struct {
	ID graphql.ID
}) (*DeleteBridgePayloadResolver, error) {
	if err := authenticateUserCanEdit(ctx); err != nil {
		return nil, err
	}

	taskType, err := bridges.ParseBridgeName(string(args.ID))
	if err != nil {
		return NewDeleteBridgePayload(nil, err), nil
	}

	orm := r.App.BridgeORM()
	bt, err := orm.FindBridge(taskType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NewDeleteBridgePayload(nil, err), nil
		}

		return nil, err
	}

	jobsUsingBridge, err := r.App.JobORM().FindJobIDsWithBridge(string(args.ID))
	if err != nil {
		return nil, err
	}
	if len(jobsUsingBridge) > 0 {
		return NewDeleteBridgePayload(nil, fmt.Errorf("bridge has jobs associated with it")), nil
	}

	if err = orm.DeleteBridgeType(&bt); err != nil {
		return nil, err
	}

	r.App.GetAuditLogger().Audit(audit.BridgeDeleted, map[string]interface{}{"name": bt.Name})
	return NewDeleteBridgePayload(&bt, nil), nil
}

func (r *Resolver) CreateP2PKey(ctx context.Context) (*CreateP2PKeyPayloadResolver, error) {
	if err := authenticateUserCanEdit(ctx); err != nil {
		return nil, err
	}

	key, err := r.App.GetKeyStore().P2P().Create()
	if err != nil {
		return nil, err
	}

	const keyType = "Ed25519"
	r.App.GetAuditLogger().Audit(audit.KeyCreated, map[string]interface{}{
		"type":         "p2p",
		"id":           key.ID(),
		"p2pPublicKey": key.PublicKeyHex(),
		"p2pPeerID":    key.PeerID(),
		"p2pType":      keyType,
	})

	return NewCreateP2PKeyPayload(key), nil
}

func (r *Resolver) DeleteP2PKey(ctx context.Context, args struct {
	ID graphql.ID
}) (*DeleteP2PKeyPayloadResolver, error) {
	if err := authenticateUserIsAdmin(ctx); err != nil {
		return nil, err
	}

	keyID, err := p2pkey.MakePeerID(string(args.ID))
	if err != nil {
		return nil, err
	}

	key, err := r.App.GetKeyStore().P2P().Delete(keyID)
	if err != nil {
		if errors.As(err, &keystore.KeyNotFoundError{}) {
			return NewDeleteP2PKeyPayload(p2pkey.KeyV2{}, err), nil
		}
		return nil, err
	}

	r.App.GetAuditLogger().Audit(audit.KeyDeleted, map[string]interface{}{
		"type": "p2p",
		"id":   args.ID,
	})

	return NewDeleteP2PKeyPayload(key, nil), nil
}

func (r *Resolver) CreateVRFKey(ctx context.Context) (*CreateVRFKeyPayloadResolver, error) {
	if err := authenticateUserCanEdit(ctx); err != nil {
		return nil, err
	}

	key, err := r.App.GetKeyStore().VRF().Create()
	if err != nil {
		return nil, err
	}

	r.App.GetAuditLogger().Audit(audit.KeyCreated, map[string]interface{}{
		"type":                "vrf",
		"id":                  key.ID(),
		"vrfPublicKey":        key.PublicKey,
		"vrfPublicKeyAddress": key.PublicKey.Address(),
	})

	return NewCreateVRFKeyPayloadResolver(key), nil
}

func (r *Resolver) DeleteVRFKey(ctx context.Context, args struct {
	ID graphql.ID
}) (*DeleteVRFKeyPayloadResolver, error) {
	if err := authenticateUserIsAdmin(ctx); err != nil {
		return nil, err
	}

	key, err := r.App.GetKeyStore().VRF().Delete(string(args.ID))
	if err != nil {
		if errors.Is(errors.Cause(err), keystore.ErrMissingVRFKey) {
			return NewDeleteVRFKeyPayloadResolver(vrfkey.KeyV2{}, err), nil
		}
		return nil, err
	}

	r.App.GetAuditLogger().Audit(audit.KeyDeleted, map[string]interface{}{
		"type": "vrf",
		"id":   args.ID,
	})

	return NewDeleteVRFKeyPayloadResolver(key, nil), nil
}

func (r *Resolver) UpdateUserPassword(ctx context.Context, args struct {
	Input UpdatePasswordInput
}) (*UpdatePasswordPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	session, ok := webauth.GetGQLAuthenticatedSession(ctx)
	if !ok {
		return nil, errors.New("couldn't retrieve user session")
	}

	dbUser, err := r.App.AuthenticationProvider().FindUser(session.User.Email)
	if err != nil {
		return nil, err
	}

	if !utils.CheckPasswordHash(args.Input.OldPassword, dbUser.HashedPassword) {
		r.App.GetAuditLogger().Audit(audit.PasswordResetAttemptFailedMismatch, map[string]interface{}{"user": dbUser.Email})

		return NewUpdatePasswordPayload(nil, map[string]string{
			"oldPassword": "old password does not match",
		}), nil
	}

	if err = r.App.AuthenticationProvider().ClearNonCurrentSessions(session.SessionID); err != nil {
		return nil, clearSessionsError{}
	}

	err = r.App.AuthenticationProvider().SetPassword(&dbUser, args.Input.NewPassword)
	if err != nil {
		return nil, failedPasswordUpdateError{}
	}

	r.App.GetAuditLogger().Audit(audit.PasswordResetSuccess, map[string]interface{}{"user": dbUser.Email})
	return NewUpdatePasswordPayload(session.User, nil), nil
}

func (r *Resolver) SetSQLLogging(ctx context.Context, args struct {
	Input struct{ Enabled bool }
}) (*SetSQLLoggingPayloadResolver, error) {
	if err := authenticateUserIsAdmin(ctx); err != nil {
		return nil, err
	}

	r.App.GetConfig().SetLogSQL(args.Input.Enabled)

	if args.Input.Enabled {
		r.App.GetAuditLogger().Audit(audit.ConfigSqlLoggingEnabled, map[string]interface{}{})
	} else {
		r.App.GetAuditLogger().Audit(audit.ConfigSqlLoggingDisabled, map[string]interface{}{})
	}

	return NewSetSQLLoggingPayload(args.Input.Enabled), nil
}

func (r *Resolver) CreateAPIToken(ctx context.Context, args struct {
	Input struct{ Password string }
}) (*CreateAPITokenPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	session, ok := webauth.GetGQLAuthenticatedSession(ctx)
	if !ok {
		return nil, errors.New("Failed to obtain current user from context")
	}
	dbUser, err := r.App.AuthenticationProvider().FindUser(session.User.Email)
	if err != nil {
		return nil, err
	}

	err = r.App.AuthenticationProvider().TestPassword(dbUser.Email, args.Input.Password)
	if err != nil {
		r.App.GetAuditLogger().Audit(audit.APITokenCreateAttemptPasswordMismatch, map[string]interface{}{"user": dbUser.Email})

		return NewCreateAPITokenPayload(nil, map[string]string{
			"password": "incorrect password",
		}), nil
	}

	newToken, err := r.App.AuthenticationProvider().CreateAndSetAuthToken(&dbUser)
	if err != nil {
		return nil, err
	}

	r.App.GetAuditLogger().Audit(audit.APITokenCreated, map[string]interface{}{"user": dbUser.Email})
	return NewCreateAPITokenPayload(newToken, nil), nil
}

func (r *Resolver) DeleteAPIToken(ctx context.Context, args struct {
	Input struct{ Password string }
}) (*DeleteAPITokenPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	session, ok := webauth.GetGQLAuthenticatedSession(ctx)
	if !ok {
		return nil, errors.New("Failed to obtain current user from context")
	}
	dbUser, err := r.App.AuthenticationProvider().FindUser(session.User.Email)
	if err != nil {
		return nil, err
	}

	err = r.App.AuthenticationProvider().TestPassword(dbUser.Email, args.Input.Password)
	if err != nil {
		r.App.GetAuditLogger().Audit(audit.APITokenDeleteAttemptPasswordMismatch, map[string]interface{}{"user": dbUser.Email})

		return NewDeleteAPITokenPayload(nil, map[string]string{
			"password": "incorrect password",
		}), nil
	}

	err = r.App.AuthenticationProvider().DeleteAuthToken(&dbUser)
	if err != nil {
		return nil, err
	}

	r.App.GetAuditLogger().Audit(audit.APITokenDeleted, map[string]interface{}{"user": dbUser.Email})

	return NewDeleteAPITokenPayload(&auth.Token{
		AccessKey: dbUser.TokenKey.String,
	}, nil), nil
}

func (r *Resolver) CreateJob(ctx context.Context, args struct {
	Input struct {
		TOML string
	}
}) (*CreateJobPayloadResolver, error) {
	if err := authenticateUserCanEdit(ctx); err != nil {
		return nil, err
	}

	jbt, err := job.ValidateSpec(args.Input.TOML)
	if err != nil {
		return NewCreateJobPayload(r.App, nil, map[string]string{
			"TOML spec": errors.Wrap(err, "failed to parse TOML").Error(),
		}), nil
	}

	var jb job.Job
	switch jbt {
	case job.BlockhashStore:
		jb, err = blockhashstore.ValidatedSpec(args.Input.TOML)
	case job.VRF:
		jb, err = vrfcommon.ValidatedVRFSpec(args.Input.TOML)
	case job.Webhook:
		jb, err = webhook.ValidatedWebhookSpec(args.Input.TOML, r.App.GetExternalInitiatorManager())
	case job.Gateway:
		jb, err = gateway.ValidatedGatewaySpec(args.Input.TOML)
	default:
		return NewCreateJobPayload(r.App, nil, map[string]string{
			"Job Type": fmt.Sprintf("unknown job type: %s", jbt),
		}), nil
	}
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = r.App.AddJobV2(ctx, &jb)
	if err != nil {
		return nil, err
	}

	jbj, _ := json.Marshal(jb)
	r.App.GetAuditLogger().Audit(audit.JobCreated, map[string]interface{}{"job": string(jbj)})

	return NewCreateJobPayload(r.App, &jb, nil), nil
}

func (r *Resolver) DeleteJob(ctx context.Context, args struct {
	ID graphql.ID
}) (*DeleteJobPayloadResolver, error) {
	if err := authenticateUserCanEdit(ctx); err != nil {
		return nil, err
	}

	id, err := stringutils.ToInt32(string(args.ID))
	if err != nil {
		return nil, err
	}

	j, err := r.App.JobORM().FindJobWithoutSpecErrors(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NewDeleteJobPayload(r.App, nil, err), nil
		}

		return nil, err
	}

	err = r.App.DeleteJob(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NewDeleteJobPayload(r.App, nil, err), nil
		}

		return nil, err
	}

	r.App.GetAuditLogger().Audit(audit.JobDeleted, map[string]interface{}{"id": args.ID})
	return NewDeleteJobPayload(r.App, &j, nil), nil
}

func (r *Resolver) DismissJobError(ctx context.Context, args struct {
	ID graphql.ID
}) (*DismissJobErrorPayloadResolver, error) {
	if err := authenticateUserCanEdit(ctx); err != nil {
		return nil, err
	}

	id, err := stringutils.ToInt64(string(args.ID))
	if err != nil {
		return nil, err
	}

	specErr, err := r.App.JobORM().FindSpecError(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NewDismissJobErrorPayload(nil, err), nil
		}

		return nil, err
	}

	err = r.App.JobORM().DismissError(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NewDismissJobErrorPayload(nil, err), nil
		}

		return nil, err
	}

	r.App.GetAuditLogger().Audit(audit.JobErrorDismissed, map[string]interface{}{"id": args.ID})
	return NewDismissJobErrorPayload(&specErr, nil), nil
}

func (r *Resolver) RunJob(ctx context.Context, args struct {
	ID graphql.ID
}) (*RunJobPayloadResolver, error) {
	if err := authenticateUserCanRun(ctx); err != nil {
		return nil, err
	}

	jobID, err := stringutils.ToInt32(string(args.ID))
	if err != nil {
		return nil, err
	}

	jobRunID, err := r.App.RunJobV2(ctx, jobID, nil)
	if err != nil {
		if errors.Is(err, webhook.ErrJobNotExists) {
			return NewRunJobPayload(nil, r.App, err), nil
		}

		return nil, err
	}

	plnRun, err := r.App.PipelineORM().FindRun(jobRunID)
	if err != nil {
		return nil, err
	}

	r.App.GetAuditLogger().Audit(audit.JobRunSet, map[string]interface{}{"jobID": args.ID, "jobRunID": jobRunID, "planRunID": plnRun})
	return NewRunJobPayload(&plnRun, r.App, nil), nil
}

func (r *Resolver) SetGlobalLogLevel(ctx context.Context, args struct {
	Level LogLevel
}) (*SetGlobalLogLevelPayloadResolver, error) {
	if err := authenticateUserIsAdmin(ctx); err != nil {
		return nil, err
	}

	var lvl zapcore.Level
	logLvl := FromLogLevel(args.Level)

	err := lvl.UnmarshalText([]byte(logLvl))
	if err != nil {
		return NewSetGlobalLogLevelPayload("", map[string]string{
			"level": "invalid log level",
		}), nil
	}

	if err := r.App.SetLogLevel(lvl); err != nil {
		return nil, err
	}

	r.App.GetAuditLogger().Audit(audit.GlobalLogLevelSet, map[string]interface{}{"logLevel": args.Level})
	return NewSetGlobalLogLevelPayload(args.Level, nil), nil
}
