package workers

import (
	"bytes"
	"testing"
)

func TestOutputParser_Next(t *testing.T) {
	const sp = "abc"
	inputs := []string{
		"hoge" + sp + "fuga",
		"ab" + sp + "ab" + sp,
		"hoge" + sp + "fuga" + sp + "piyo" + sp + "poyo",
		"aiueoaiueoaiueo" + sp + "kakikukeko",
		"aiue" + sp + "kaki" + sp + "sasi",
		sp + sp + sp,
		"i" + sp + "j" + sp + "k",
		"a" + sp + "a" + sp + "a",
		"aiueoaiueoaiueo" + sp + "kakikukeko" + sp + "sashisuseso" + sp + "tatitsuteto",
	}
	outputs := [][]string{
		{
			"hoge",
			"fuga",
		},
		{
			"ab",
			"ab",
			"",
		},
		{
			"hoge",
			"fuga",
			"piyo",
			"poyo",
		},
		{
			"aiueoaiueoaiueo",
			"kakikukeko",
		},
		{
			"aiue",
			"kaki",
			"sasi",
		},
		{
			"",
			"",
			"",
			"",
		},
		{
			"i",
			"j",
			"k",
		},
		{
			"a",
			"a",
			"a",
		},
		{
			"aiueoaiueoaiueo",
			"kakikukeko",
			"sashisuseso",
			"tatitsuteto",
		},
	}

	for i := 0; i < 5; i++ {
		r := bytes.NewReader([]byte(inputs[i]))
		op := newOutputParser(r, sp)

		for j, out := range outputs[i] {
			nx, str, err := op.Next()
			if err != nil {
				t.Error(err)
				break
			}
			if !nx && j != len(outputs[i])-1 {
				t.Errorf("has next! on case %v", i)
				break
			}
			if out != str {
				t.Errorf("output invalid on case %v: output->%v actual->%v", i, out, str)
				break
			}
		}
	}
}
