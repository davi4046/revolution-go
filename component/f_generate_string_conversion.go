package component

import (
	"fmt"
	"revolution/randutil"
	"strings"
	"text/template"
)

func generateStringConversion(src, dst, dstType string) (string, error) {

	base := strings.TrimPrefix(dstType, "[]")

	if base == "string" {
		if strings.HasPrefix(dstType, "[]") {
			return fmt.Sprintf("%s := %s", dst, src), nil
		} else {
			return fmt.Sprintf("%s := strings.Split(%s, ',')", dst, src), nil
		}
	} else {

		convMethod, ok := map[string]string{
			"int":     "strconv.ParseInt(%s, 10, 0)",
			"int8":    "strconv.ParseInt(%s, 10, 8)",
			"int16":   "strconv.ParseInt(%s, 10, 16)",
			"int32":   "strconv.ParseInt(%s, 10, 32)",
			"int64":   "strconv.ParseInt(%s, 10, 64)",
			"uint":    "strconv.ParseUint(%s, 10, 0)",
			"uint8":   "strconv.ParseUint(%s, 10, 8)",
			"uint16":  "strconv.ParseUint(%s, 10, 16)",
			"uint32":  "strconv.ParseUint(%s, 10, 32)",
			"uint64":  "strconv.ParseUint(%s, 10, 64)",
			"float32": "strconv.ParseFloat(%s, 32)",
			"float64": "strconv.ParseFloat(%s, 64)",
			"bool":    "strconv.ParseBool(%s)",
		}[base]
		if !ok {
			return "", fmt.Errorf("conversion from string to %s is not implemented", base)
		}

		stringConvTempl, err := template.New("").Parse(
			`{{.TmpVarName}}, err := {{.ConvMethod}}; if err != nil { log.Fatalln(err) }; {{.VarName}} := {{.Type}}({{.TmpVarName}})`,
		)
		if err != nil {
			return "", err
		}

		sliceConvTempl, err := template.New("").Parse(
			`var {{.VarName}} []{{.Type}}; for _, s := range strings.Split({{.Source}}, ',') { {{.LoopBody}} }`,
		)
		if err != nil {
			return "", err
		}

		tmpVarName := randutil.GetRandomString(20)

		if strings.HasPrefix(dstType, "[]") {

			convMethod = fmt.Sprintf(convMethod, "s")

			convData := struct {
				VarName, TmpVarName, ConvMethod, Type string
			}{
				VarName:    "val",
				TmpVarName: tmpVarName,
				ConvMethod: convMethod,
				Type:       base,
			}

			var builder strings.Builder
			if err := stringConvTempl.Execute(&builder, convData); err != nil {
				return "", err
			}

			convBody := builder.String()
			loopBody := fmt.Sprintf("%[1]s; %[2]s = append(%[2]s, %[3]s)",
				convBody,
				dst,
				"val",
			)

			listConvData := struct {
				VarName, Type, Source, LoopBody string
			}{
				VarName:  dst,
				Type:     base,
				Source:   src,
				LoopBody: loopBody,
			}

			builder.Reset()
			if err := sliceConvTempl.Execute(&builder, listConvData); err != nil {
				return "", err
			}

			return builder.String(), nil

		} else {

			convMethod = fmt.Sprintf(convMethod, src)

			data := struct {
				VarName, TmpVarName, ConvMethod, Type string
			}{
				VarName:    dst,
				TmpVarName: tmpVarName,
				ConvMethod: convMethod,
				Type:       base,
			}

			var builder strings.Builder
			if err := stringConvTempl.Execute(&builder, data); err != nil {
				return "", err
			}

			return builder.String(), nil
		}
	}
}
