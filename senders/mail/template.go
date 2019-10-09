package mail

const defaultTemplate = `
<!doctype html>
<html>

<head>
    <meta name="viewport" content="width=device-width">
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
    <title>Moira Alert</title>
    <style media="all" type="text/css">
        tr.OK {
            color: #228007;
        }

        tr.WARN {
            color: #D97E00;
        }

        tr.ERROR {
            color: #CE0014;
        }

        tr.NODATA {
            color: #CE0014;
        }

        tr.EXCEPTION {
            color: #CE0014;
        }

        tr.TEST {
            color: #228007;
        }

        @media only screen and (max-width: 820px) {
            table[class=body] h1,
            table[class=body] h2,
            table[class=body] h3,
            table[class=body] h4 {
                font-weight: 600 !important;
            }

            table[class=body] h1 {
                font-size: 22px !important;
            }

            table[class=body] h2 {
                font-size: 18px !important;
            }

            table[class=body] h3 {
                font-size: 16px !important;
            }

            table[class=body] .content,
            table[class=body] .wrapper {
                padding: 10px !important;
            }

            table[class=body] .container {
                padding: 0 !important;
                width: 100% !important;
            }

            table[class=body] .btn table,
            table[class=body] .btn a {
                width: 100% !important;
            }
        }
    </style>

</head>

<body style="margin: 0; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 14px; height: 100% !important; line-height: 1.6em; -webkit-font-smoothing: antialiased; padding: 0; -ms-text-size-adjust: 100%; -webkit-text-size-adjust: 100%; width: 100% !important; background-color: #f6f6f6;">

    <table class="body" style="box-sizing: border-box; border-collapse: separate !important; mso-table-lspace: 0pt; mso-table-rspace: 0pt; width: 100%; background-color: #f6f6f6;"
        width="100%" bgcolor="#f6f6f6">
        <tr>
            <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500;"
                valign="top"></td>
            <td class="container" style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500; display: block; Margin: 0 auto !important; max-width: 780px; padding: 10px; width: 780px;"
                width="780" valign="top">
                <div class="content" style="box-sizing: border-box; display: block; margin: 0 auto; max-width: 780px; padding: 10px;">
                    <table class="main" style="box-sizing: border-box; border-collapse: separate !important; mso-table-lspace: 0pt; mso-table-rspace: 0pt; width: 100%; background: #ffffff; border: 1px solid #e9e9e9; border-radius: 3px;"
                        width="100%">
                        <tr>
                            <td class="wrapper" style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500; padding: 30px;"
                                valign="top">
                                <table style="box-sizing: border-box; border-collapse: separate !important; mso-table-lspace: 0pt; mso-table-rspace: 0pt; width: 100%;"
                                    width="100%">
                                    <tr>
                                        <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500;"
                                            valign="top">
                                            <h1 class="align-left h1-nopadding" style="text-align: left; color: #333333 !important; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-weight: 600; line-height: 1.4em; margin: 0 0 5px 0; font-size: 30px;">
                                                {{if .TriggerName}} {{ .TriggerState }}! {{ .TriggerName }} {{else}} TEST notification {{end}}
                                            </h1>
                                            <h4 class="align-left" style="text-align: left; color: #9B9B9B !important; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-weight: 600; line-height: 1.4em; margin: 0 0 5px 0; font-size: 16px;">
                                                {{if .Tags}} {{ .Tags }} {{else}} [test] {{end}}
                                            </h4>
                                            {{ if .Throttled}}
                                            <table class="notice-wrapper" cellpadding="0" cellspacing="0" style="box-sizing: border-box; border-collapse: separate !important; mso-table-lspace: 0pt; mso-table-rspace: 0pt; width: 100%;"
                                                width="100%">
                                                <tr>
                                                    <td class="notice-spacer" style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500; padding: 5px 0 5px 0;"
                                                        valign="top">
                                                        <table class="notice notice-throttling " cellpadding="0" cellspacing="0" style="box-sizing: border-box; border-collapse: separate !important; mso-table-lspace: 0pt; mso-table-rspace: 0pt; width: 100%;"
                                                            width="100%">
                                                            <tr>
                                                                <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; vertical-align: top; font-size: 18px; font-weight: 400; line-height: 1.6; border: 1px solid #D0021B; color: #D0021B; padding: 12px 12px;"
                                                                    valign="top">Please, fix your system or tune this trigger to generate
                                                                    less events.
                                                                </td>
                                                            </tr>
                                                        </table>
                                                    </td>
                                                </tr>
                                            </table>
                                            {{end}}
                                            <table class="divider-wrapper" style="box-sizing: border-box; border-collapse: separate !important; mso-table-lspace: 0pt; mso-table-rspace: 0pt; width: 100%;"
                                                width="100%">
                                                <tr>
                                                    <td class="divider-spacer" style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500; padding: 5px 0 5px 0;"
                                                        valign="top">
                                                        <table class="divider divider- " cellpadding="0" cellspacing="0" style="box-sizing: border-box; border-collapse: separate !important; mso-table-lspace: 0pt; mso-table-rspace: 0pt; width: 100%;"
                                                            width="100%">
                                                            <tr>
                                                                <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; vertical-align: top; font-weight: 500; font-size: 0; border-top: 1px solid #ccc; line-height: 0; height: 1px; margin: 0; padding: 0;"
                                                                    valign="top"></td>
                                                            </tr>
                                                        </table>
                                                    </td>
                                                </tr>
                                            </table>
                                        </td>
                                    </tr>
                                    <tr>
                                        <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500;"
                                            valign="top">
                                            <table style="box-sizing: border-box; border-collapse: separate !important; mso-table-lspace: 0pt; mso-table-rspace: 0pt; width: 100%;"
                                                width="100%">
                                                <tbody>
                                                    <tr>
                                                        <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500;"
                                                            valign="top">
                                                            <h5 class="align-left h5-nopadding" style="font-size: 12px; font-weight: 700; color: #9B9B9B !important; margin-bottom: 0 !important; margin-top: 0 !important; text-align: left;">
                                                                Timestamp</h5>
                                                        </td>
                                                        <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500;"
                                                            valign="top">
                                                            <h5 class="align-left h5-nopadding" style="font-size: 12px; font-weight: 700; color: #9B9B9B !important; margin-bottom: 0 !important; margin-top: 0 !important; text-align: left;">
                                                                Target</h5>
                                                        </td>
                                                        <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500;"
                                                            valign="top">
                                                            <h5 class="align-left h5-nopadding" style="font-size: 12px; font-weight: 700; color: #9B9B9B !important; margin-bottom: 0 !important; margin-top: 0 !important; text-align: left;">
                                                                Values</h5>
                                                        </td>
                                                        <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500;"
                                                            valign="top">
                                                            <h5 class="align-left h5-nopadding" style="font-size: 12px; font-weight: 700; color: #9B9B9B !important; margin-bottom: 0 !important; margin-top: 0 !important; text-align: left;">
                                                                State</h5>
                                                        </td>
                                                        <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500;"
                                                            valign="top">
                                                            <h5 class="align-left h5-nopadding" style="font-size: 12px; font-weight: 700; color: #9B9B9B !important; margin-bottom: 0 !important; margin-top: 0 !important; text-align: left;">
                                                                Note</h5>
                                                        </td>
                                                    </tr>
                                                    {{range .Items}}
                                                    <tr class="{{ .State }}">
                                                        <td class="td-width20 td-padding" style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500; width: 20%; padding-bottom: 10px; padding-right: 3px;"
                                                            width="20%" valign="top">
                                                            {{ .Timestamp }}
                                                        </td>
                                                        <td class="td-width20 td-padding" style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500; width: 20%; padding-bottom: 10px; padding-right: 3px;"
                                                            width="20%" valign="top">
                                                            {{ .Metric }}
                                                        </td>
                                                        <td class="td-width20 td-padding" style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500; width: 20%; padding-bottom: 10px; padding-right: 3px;"
                                                            width="20%" valign="top">
                                                            {{ .Values }}
                                                        </td>
                                                        <td class="td-width20 td-padding" style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500; width: 20%; padding-bottom: 10px; padding-right: 3px;"
                                                            width="20%" valign="top">
                                                            {{ .Oldstate }}-{{ .State }}
                                                        </td>
                                                        <td class="td-width20 td-padding" style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500; width: 20%; padding-bottom: 10px; padding-right: 3px;"
                                                            width="20%" valign="top">
                                                            {{ .Message }}
                                                        </td>
                                                    </tr>
                                                    {{end}}
                                                </tbody>
                                            </table>
                                        </td>
                                    </tr>
                                    {{if .Description }}
                                    <tr>
                                        <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500;"
                                            valign="top">
                                            <table class="divider-wrapper" style="box-sizing: border-box; border-collapse: separate !important; mso-table-lspace: 0pt; mso-table-rspace: 0pt; width: 100%;"
                                                width="100%">
                                                <tr>
                                                    <td class="divider-spacer" style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500; padding: 5px 0 5px 0;"
                                                        valign="top">
                                                        <table class="divider divider- " cellpadding="0" cellspacing="0" style="box-sizing: border-box; border-collapse: separate !important; mso-table-lspace: 0pt; mso-table-rspace: 0pt; width: 100%;"
                                                            width="100%">
                                                            <tr>
                                                                <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; vertical-align: top; font-weight: 500; font-size: 0; border-top: 1px solid #ccc; line-height: 0; height: 1px; margin: 0; padding: 0;"
                                                                    valign="top"></td>
                                                            </tr>
                                                        </table>
                                                    </td>
                                                </tr>
                                            </table>
                                        </td>
                                    </tr>
                                    <tr>
                                        <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500;"
                                            valign="top">
                                            <table class="notice-wrapper" cellpadding="0" cellspacing="0" style="box-sizing: border-box; border-collapse: separate !important; mso-table-lspace: 0pt; mso-table-rspace: 0pt; width: 100%;"
                                                width="100%">
                                                <tr>
                                                    <td class="notice-spacer" style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500; padding: 5px 0 5px 0;"
                                                        valign="top">
                                                        <table class="notice notice-description " cellpadding="0" cellspacing="0" style="box-sizing: border-box; border-collapse: separate !important; mso-table-lspace: 0pt; mso-table-rspace: 0pt; width: 100%;"
                                                            width="100%">
                                                            <tr>
                                                                <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; vertical-align: top; font-size: 16px; font-weight: 400; line-height: 1.6; color: #807F80; padding-bottom: 50px;"
                                                                    valign="top">
                                                                    {{ .Description }}
                                                                </td>
                                                            </tr>
                                                        </table>
                                                    </td>
                                                </tr>
                                            </table>
                                        </td>
                                    </tr>
                                    {{end}} {{if .PlotCID }}
                                    <tr>
                                        <td class="align-center" style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500; text-align: center;"
                                            valign="top" align="center">
                                            <img src="cid:{{ .PlotCID }}" alt="Trigger plot" style="-ms-interpolation-mode: bicubic; max-width: 100%;">
                                        </td>
                                    </tr>
                                    {{end}} {{ if .Link }}
                                    <tr>
                                        <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500;"
                                            valign="top">
                                            <table class="btn btn-secondary btn-expanded" cellpadding="0" cellspacing="0" style="box-sizing: border-box; border-collapse: separate !important; mso-table-lspace: 0pt; mso-table-rspace: 0pt; width: 100%;"
                                                width="100%">
                                                <tr>
                                                    <td align="center" style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500; padding-bottom: 15px;"
                                                        valign="top">
                                                        <table cellpadding="0" cellspacing="0" style="box-sizing: border-box; border-collapse: separate !important; mso-table-lspace: 0pt; mso-table-rspace: 0pt; width: 100%;"
                                                            width="100%">
                                                            <tr>
                                                                <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500; background-color: transparent; border-radius: 5px; text-align: center;"
                                                                    valign="top" bgcolor="transparent" align="center">
                                                                    <a href="{{ .Link }}" style="box-sizing: border-box; font-weight: bold; text-decoration: none; background-color: transparent; border: solid 1px #3072C4; border-radius: 5px; cursor: pointer; display: inline-block; font-size: 14px; color: #3072C4; margin: 0; padding: 12px 25px; text-transform: capitalize; border-color: #3072C4; text-align: center; width: 100%; padding-left: 0; padding-right: 0;">Open in Moira</a>
                                                                </td>
                                                            </tr>
                                                        </table>
                                                    </td>
                                                </tr>
                                            </table>
                                        </td>
                                    </tr>
                                    {{ end }}
                                </table>
                            </td>
                        </tr>
                    </table>
                </div>
            </td>
            <td style="box-sizing: border-box; font-family: 'Segoe UI', 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; vertical-align: top; font-weight: 500;"
                valign="top"></td>
        </tr>
    </table>

</body>

</html>
`
