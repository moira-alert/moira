package mail

const defaultTemplate = `
<html>
	<head>
		<style type="text/css">
			table { border-collapse: collapse; }
			table th, table td { padding: 0.5em; }
			tr.OK { background-color: #33cc99; color: white; }
			tr.WARN { background-color: #cccc32; color: white; }
			tr.ERROR { background-color: #cc0032; color: white; }
			tr.NODATA { background-color: #d3d3d3; color: black; }
			tr.EXCEPTION { background-color: #e14f4f; color: white; }
			th, td { border: 1px solid black; }
		</style>
	</head>
	<body>
		<table>
			<thead>
				<tr>
					<th>Timestamp</th>
					<th>Target</th>
					<th>Value</th>
					<th>Warn</th>
					<th>Error</th>
					<th>From</th>
					<th>To</th>
					<th>Note</th>
				</tr>
			</thead>
			<tbody>
				{{range .Items}}
				<tr class="{{ .State }}">
					<td>{{ .Timestamp }}</td>
					<td>{{ .Metric }}</td>
					<td>{{ .Value }}</td>
					<td>{{ .WarnValue }}</td>
					<td>{{ .ErrorValue }}</td>
					<td>{{ .Oldstate }}</td>
					<td>{{ .State }}</td>
					<td>{{ .Message }}</td>
				</tr>
				{{end}}
			</tbody>
		</table>
		<p>Description: {{ .Description }}</p>
		<p><a href="{{ .Link }}">{{ .Link }}</a></p>
		{{if .Throttled}}
		<p>Please, <b>fix your system or tune this trigger</b> to generate less events.</p>
		{{end}}
	</body>
</html>
`

