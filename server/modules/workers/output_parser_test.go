package workers

import (
	"bytes"
	"testing"
)

func TestOutputParser_Next(t *testing.T) {
	sp := []string{
		"abc",
		"abc",
		"abc",
		"abc",
		"abc",
		"abc",
		"abc",
		"abc",
		"abc",
		"XucMvC62n7h8u9ORt7_6LQ==",
	}
	inputs := []string{
		"hoge" + sp[0] + "fuga",
		"ab" + sp[1] + "ab" + sp[1],
		"hoge" + sp[2] + "fuga" + sp[2] + "piyo" + sp[2] + "poyo",
		"aiueoaiueoaiueo" + sp[3] + "kakikukeko",
		"aiue" + sp[4] + "kaki" + sp[4] + "sasi",
		sp[5] + sp[5] + sp[5],
		"i" + sp[6] + "j" + sp[6] + "k",
		"a" + sp[7] + "a" + sp[7] + "a",
		"aiueoaiueoaiueo" + sp[8] + "kakikukeko" + sp[8] + "sashisuseso" + sp[8] + "tatitsuteto",
		`1649982674
XucMvC62n7h8u9ORt7_6LQ==0XucMvC62n7h8u9ORt7_6LQ==630985090
XucMvC62n7h8u9ORt7_6LQ==0XucMvC62n7h8u9ORt7_6LQ==964205429
XucMvC62n7h8u9ORt7_6LQ==0XucMvC62n7h8u9ORt7_6LQ==1781341663
XucMvC62n7h8u9ORt7_6LQ==0XucMvC62n7h8u9ORt7_6LQ==825511409
XucMvC62n7h8u9ORt7_6LQ==0XucMvC62n7h8u9ORt7_6LQ==1487948132
XucMvC62n7h8u9ORt7_6LQ==0XucMvC62n7h8u9ORt7_6LQ==653541092
XucMvC62n7h8u9ORt7_6LQ==0XucMvC62n7h8u9ORt7_6LQ==1230221322
XucMvC62n7h8u9ORt7_6LQ==0XucMvC62n7h8u9ORt7_6LQ==1025796403
XucMvC62n7h8u9ORt7_6LQ==0XucMvC62n7h8u9ORt7_6LQ==1241727600
XucMvC62n7h8u9ORt7_6LQ==0XucMvC62n7h8u9ORt7_6LQ==
`,
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
		{
			"1649982674\n",
			"0",
			"630985090\n",
			"0",
			"964205429\n",
			"0",
			"1781341663\n",
			"0",
			"825511409\n",
			"0",
			"1487948132\n",
			"0",
			"653541092\n",
			"0",
			"1230221322\n",
			"0",
			"1025796403\n",
			"0",
			"1241727600\n",
			"0",
			"\n",
		},
	}

	for i := 0; i < len(inputs); i++ {
		r := bytes.NewReader([]byte(inputs[i]))
		op := newReaderParser(r, sp[i])

		for j, out := range outputs[i] {
			if j == 12 {
				t.Log("hoge")
			}
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
