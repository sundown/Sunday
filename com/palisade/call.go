package palisade

import (
	"os"
	"sundown/solution/oversight"

	"github.com/alecthomas/participle/v2"
)

var Parser = participle.MustBuild(&PalisadeResult{}, participle.UseLookahead(4), participle.Unquote())

func Begin(rt *oversight.Runtime) *PalisadeResult {
	prog := PalisadeResult{}
	oversight.Verbose("Init palisade")
	r, err := os.Open(rt.File)
	defer r.Close()

	if err != nil {
		oversight.Error(err.Error()).Exit()
	}

	err = Parser.Parse(rt.File, r, &prog)

	if err != nil {
		panic(err)
	}

	return &prog
}
