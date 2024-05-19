package gojq

import (
	"encoding/json"
	"fmt"

	"github.com/bartdeboer/pipeline"
	"github.com/itchyny/gojq"
)

// JQ executes query on the pipe's contents (presumed to be JSON), producing
// the result. An invalid query will set the appropriate error on the pipe.
//
// The exact dialect of JQ supported is that provided by
// [github.com/itchyny/gojq], whose documentation explains the differences
// between it and standard JQ.
func JQ(query string) pipeline.Program {
	p := pipeline.NewBaseProgram()
	p.StartFn = func() error {
		q, err := gojq.Parse(query)
		if err != nil {
			return err
		}
		var input interface{}
		err = json.NewDecoder(p.Stdin).Decode(&input)
		if err != nil {
			return err
		}
		iter := q.Run(input)
		for {
			v, ok := iter.Next()
			if !ok {
				return nil
			}
			if err, ok := v.(error); ok {
				return err
			}
			result, err := gojq.Marshal(v)
			if err != nil {
				return err
			}
			fmt.Fprintln(p.Stdout, string(result))
		}
	}
	return p
}
