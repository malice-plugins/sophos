package main

const tpl = `#### Sophos
{{- with .Results }}
| Infected      | Result      | Engine      | Updated      |
|:-------------:|:-----------:|:-----------:|:------------:|
| {{.Infected}} | {{.Result}} | {{.Engine}} | {{.Updated}} |
{{ end -}}
`

// func printMarkDownTable(sophos Sophos) {

// 	fmt.Println("#### Sophos")
// 	table := clitable.New([]string{"Infected", "Result", "Engine", "Updated"})
// 	table.AddRow(map[string]interface{}{
// 		"Infected": sophos.Results.Infected,
// 		"Result":   sophos.Results.Result,
// 		"Engine":   sophos.Results.Engine,
// 		"Updated":  sophos.Results.Updated,
// 	})
// 	table.Markdown = true
// 	table.Print()
// }
