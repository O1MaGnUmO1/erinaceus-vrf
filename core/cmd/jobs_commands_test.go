package cmd_test

import (
	_ "embed"
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/cmd"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/cltest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/erinaceus"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/job"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/store/models"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/web/presenters"
)

func TestJob_FriendlyCreatedAt(t *testing.T) {
	t.Parallel()

	now := time.Now()

	testCases := []struct {
		name   string
		job    *cmd.JobPresenter
		result string
	}{
		{
			"gets the vrf spec created at timestamp",
			&cmd.JobPresenter{
				JobResource: presenters.JobResource{
					Type: presenters.VRFJobSpec,
					VRFSpec: &presenters.VRFSpec{
						CreatedAt: now,
					},
				},
			},
			now.Format(time.RFC3339),
		},
		{
			"invalid type",
			&cmd.JobPresenter{
				JobResource: presenters.JobResource{
					Type: "invalid type",
				},
			},
			"unknown",
		},
		{
			"no spec exists",
			&cmd.JobPresenter{
				JobResource: presenters.JobResource{
					Type: presenters.DirectRequestJobSpec,
				},
			},
			"N/A",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.result, tc.job.FriendlyCreatedAt())
		})
	}
}

func TestShell_ListFindJobs(t *testing.T) {
	t.Parallel()

	app := startNewApplicationV2(t, func(c *erinaceus.Config, s *erinaceus.Secrets) {
		c.EVM[0].Enabled = ptr(true)
	})
	client, r := app.NewShellAndRenderer()

	// Create the job
	fs := flag.NewFlagSet("", flag.ExitOnError)
	flagSetApplyFromAction(client.CreateJob, fs, "")

	err := client.CreateJob(cli.NewContext(nil, fs, nil))
	require.NoError(t, err)
	require.Len(t, r.Renders, 1)

	require.Nil(t, client.ListJobs(cltest.EmptyCLIContext()))
	jobs := *r.Renders[1].(*cmd.JobPresenters)
	require.Equal(t, 1, len(jobs))
}

func TestShell_ShowJob(t *testing.T) {
	t.Parallel()

	app := startNewApplicationV2(t, func(c *erinaceus.Config, s *erinaceus.Secrets) {
		c.EVM[0].Enabled = ptr(true)
	})
	client, r := app.NewShellAndRenderer()

	// Create the job
	fs := flag.NewFlagSet("", flag.ExitOnError)
	flagSetApplyFromAction(client.CreateJob, fs, "")

	err := client.CreateJob(cli.NewContext(nil, fs, nil))
	require.NoError(t, err)
	require.Len(t, r.Renders, 1)
	createOutput, ok := r.Renders[0].(*cmd.JobPresenter)
	require.True(t, ok, "Expected Renders[0] to be *cmd.JobPresenter, got %T", r.Renders[0])

	set := flag.NewFlagSet("test", 0)
	err = set.Parse([]string{createOutput.ID})
	require.NoError(t, err)
	c := cli.NewContext(nil, set, nil)

	require.NoError(t, client.ShowJob(c))
	job := *r.Renders[0].(*cmd.JobPresenter)
	assert.Equal(t, createOutput.ID, job.ID)
}

func TestShell_CreateJobV2(t *testing.T) {
	t.Parallel()

	app := startNewApplicationV2(t, func(c *erinaceus.Config, s *erinaceus.Secrets) {
		c.Database.Listener.FallbackPollInterval = models.MustNewDuration(100 * time.Millisecond)
		c.EVM[0].Enabled = ptr(true)
		c.EVM[0].NonceAutoSync = ptr(false)
		c.EVM[0].BalanceMonitor.Enabled = ptr(false)
		c.EVM[0].GasEstimator.Mode = ptr("FixedPrice")
	}, func(opts *startOptions) {
	})
	client, r := app.NewShellAndRenderer()

	requireJobsCount(t, app.JobORM(), 0)

	fs := flag.NewFlagSet("", flag.ExitOnError)
	flagSetApplyFromAction(client.CreateJob, fs, "")

	err := client.CreateJob(cli.NewContext(nil, fs, nil))
	require.NoError(t, err)

	requireJobsCount(t, app.JobORM(), 1)

	output := *r.Renders[0].(*cmd.JobPresenter)
	assert.Equal(t, presenters.JobSpecType("offchainreporting"), output.Type)
	assert.Equal(t, uint32(1), output.SchemaVersion)
}

func TestShell_DeleteJob(t *testing.T) {
	t.Parallel()

	app := startNewApplicationV2(t, func(c *erinaceus.Config, s *erinaceus.Secrets) {
		c.Database.Listener.FallbackPollInterval = models.MustNewDuration(100 * time.Millisecond)
		c.EVM[0].Enabled = ptr(true)
		c.EVM[0].NonceAutoSync = ptr(false)
		c.EVM[0].BalanceMonitor.Enabled = ptr(false)
		c.EVM[0].GasEstimator.Mode = ptr("FixedPrice")
	})
	client, r := app.NewShellAndRenderer()

	// Create the job
	fs := flag.NewFlagSet("", flag.ExitOnError)
	flagSetApplyFromAction(client.CreateJob, fs, "")

	err := client.CreateJob(cli.NewContext(nil, fs, nil))
	require.NoError(t, err)
	require.NotEmpty(t, r.Renders)

	output := *r.Renders[0].(*cmd.JobPresenter)

	requireJobsCount(t, app.JobORM(), 1)

	jobs, _, err := app.JobORM().FindJobs(0, 1000)
	require.NoError(t, err)
	jobID := jobs[0].ID
	cltest.AwaitJobActive(t, app.JobSpawner(), jobID, 3*time.Second)

	// Must supply job id
	set := flag.NewFlagSet("test", 0)
	flagSetApplyFromAction(client.DeleteJob, set, "")
	c := cli.NewContext(nil, set, nil)
	require.Equal(t, "must pass the job id to be archived", client.DeleteJob(c).Error())

	set = flag.NewFlagSet("test", 0)
	flagSetApplyFromAction(client.DeleteJob, set, "")

	require.NoError(t, set.Parse([]string{output.ID}))

	c = cli.NewContext(nil, set, nil)
	require.NoError(t, client.DeleteJob(c))

	requireJobsCount(t, app.JobORM(), 0)
}

func requireJobsCount(t *testing.T, orm job.ORM, expected int) {
	jobs, _, err := orm.FindJobs(0, 1000)
	require.NoError(t, err)
	require.Len(t, jobs, expected)
}
