package pipeline

import (
	"github.com/smartcontractkit/chainlink/v2/core/chains/legacyevm"
)

const (
	DotStr = `
        // data source 1
        ds1          [type=bridge name=voter_turnout];
        ds1_parse    [type=jsonparse path="one,two"];
        ds1_multiply [type=multiply times=1.23];

        // data source 2
        ds2          [type=http method=GET url="https://chain.link/voter_turnout/USA-2020" requestData=<{"hi": "hello"}>];
        ds2_parse    [type=jsonparse path="three,four"];
        ds2_multiply [type=multiply times=4.56];

        ds1 -> ds1_parse -> ds1_multiply;
        ds2 -> ds2_parse -> ds2_multiply -> answer1;

        answer1 [type=median	index=0 input1="$(ds1_multiply)" input2="$(ds2_multiply)"];
        answer2 [type=bridge 	name=election_winner	index=1];
    `
)

func (t *ETHCallTask) HelperSetDependencies(legacyChains legacyevm.LegacyChainContainer, config Config, specGasLimit *uint32, jobType string) {
	t.legacyChains = legacyChains
	t.config = config
	t.specGasLimit = specGasLimit
	t.jobType = jobType
}