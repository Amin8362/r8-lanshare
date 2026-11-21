package main

import "html/template"

type FileItem struct {
	Name    string
	Href    string
	SizeStr string
	IsDir   bool
}

type PageData struct {
	Version     string
	Title       string
	SingleFile  bool
	SingleItem  FileItem
	AllowUpload bool
	Items       []FileItem
	Notice      string
	HasParent   bool
	ParentHref  string
}

type PinPageData struct {
	Version string
	Error   string
}

var pageTmpl = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>R8Force LANShare</title>
  <style>
    :root {
      --bg: #0b1114;
      --card: #121c21;
      --border: #22343d;
      --gold: #eab308;
      --cyan: #06b6d4;
      --text: #e2e8f0;
      --muted: #94a3b8;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      background: var(--bg);
      color: var(--text);
      padding: 16px;
      max-width: 760px;
      margin: 0 auto;
    }
    .header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 14px 16px;
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 14px;
      margin-bottom: 14px;
    }
    .brand { font-weight: 800; color: var(--gold); letter-spacing: 0.5px; }
    .ver { font-size: 12px; color: var(--muted); }
    .notice {
      background: rgba(6, 182, 212, 0.14);
      border: 1px solid var(--cyan);
      color: #67e8f9;
      padding: 10px 14px;
      border-radius: 12px;
      margin-bottom: 14px;
      font-size: 14px;
    }
    .card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 14px;
      padding: 16px;
      margin-bottom: 14px;
    }
    .title {
      font-size: 14px;
      color: var(--muted);
      margin-bottom: 12px;
      word-break: break-all;
    }
    .btn {
      display: inline-block;
      background: var(--gold);
      color: #0b1114;
      font-weight: 700;
      text-decoration: none;
      border: none;
      border-radius: 10px;
      padding: 11px 18px;
      cursor: pointer;
      font-size: 14px;
    }
    .btn-cyan { background: var(--cyan); color: #0b1114; }
    .upload-form {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      align-items: center;
    }
    input[type="file"] {
      flex: 1;
      min-width: 180px;
      color: var(--text);
      background: var(--bg);
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 8px;
    }
    .list { list-style: none; }
    .item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 11px 8px;
      border-bottom: 1px solid var(--border);
      gap: 12px;
    }
    .item:last-child { border-bottom: none; }
    .item a {
      color: var(--text);
      text-decoration: none;
      font-weight: 500;
      word-break: break-all;
      flex: 1;
    }
    .item a:hover { color: var(--cyan); }
    .size { font-size: 12px; color: var(--muted); white-space: nowrap; }
    .single-box { text-align: center; padding: 14px 4px; }
    .single-name { font-size: 18px; font-weight: 700; margin-bottom: 6px; word-break: break-all; }
    .single-size { font-size: 13px; color: var(--muted); margin-bottom: 16px; }
    .footer { text-align: center; font-size: 11px; color: var(--muted); margin-top: 16px; }
  </style>
</head>
<body>
  <div class="header">
    <div class="brand">R8FORCE LANSHARE</div>
    <div class="ver">v{{.Version}}</div>
  </div>

  {{if .Notice}}
  <div class="notice">{{.Notice}}</div>
  {{end}}

  {{if .AllowUpload}}
  <div class="card">
    <div class="title">Upload files to host</div>
    <form class="upload-form" method="POST" action="/upload" enctype="multipart/form-data">
      <input type="file" name="file" multiple required />
      <button type="submit" class="btn btn-cyan">Upload</button>
    </form>
  </div>
  {{end}}

  <div class="card">
    {{if .SingleFile}}
    <div class="single-box">
      <div class="single-name">{{.SingleItem.Name}}</div>
      <div class="single-size">{{.SingleItem.SizeStr}}</div>
      <a class="btn" href="{{.SingleItem.Href}}">Download File</a>
    </div>
    {{else}}
    <div class="title">Path: {{.Title}}</div>
    <ul class="list">
      {{if .HasParent}}
      <li class="item">
        <a href="{{.ParentHref}}">.. (Parent Directory)</a>
        <span class="size">DIR</span>
      </li>
      {{end}}
      {{range .Items}}
      <li class="item">
        <a href="{{.Href}}">{{if .IsDir}}[DIR] {{end}}{{.Name}}</a>
        <span class="size">{{.SizeStr}}</span>
      </li>
      {{else}}
      <li class="item"><span class="size">Directory is empty.</span></li>
      {{end}}
    </ul>
    {{end}}
  </div>

  <div class="footer">R8Force • Local Network Transfer • No Internet Required</div>
</body>
</html>`))

var pinTmpl = template.Must(template.New("pin").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>PIN Required • R8Force LANShare</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      background: #0b1114;
      color: #e2e8f0;
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 100vh;
      margin: 0;
      padding: 16px;
    }
    .box {
      background: #121c21;
      border: 1px solid #22343d;
      border-radius: 16px;
      padding: 24px;
      width: 100%;
      max-width: 340px;
      text-align: center;
    }
    h2 { color: #eab308; margin: 0 0 8px 0; font-size: 18px; }
    p { color: #94a3b8; font-size: 13px; margin: 0 0 16px 0; }
    .err { color: #f87171; font-size: 13px; margin-bottom: 12px; }
    input {
      width: 100%;
      box-sizing: border-box;
      padding: 12px;
      border-radius: 10px;
      border: 1px solid #22343d;
      background: #0b1114;
      color: #ffffff;
      font-size: 18px;
      text-align: center;
      letter-spacing: 4px;
      margin-bottom: 12px;
    }
    button {
      width: 100%;
      padding: 12px;
      border: none;
      border-radius: 10px;
      background: #eab308;
      color: #0b1114;
      font-weight: 800;
      font-size: 15px;
      cursor: pointer;
    }
  </style>
</head>
<body>
  <div class="box">
    <h2>R8FORCE LANSHARE</h2>
    <p>Enter the PIN shown in the terminal (v{{.Version}})</p>
    {{if .Error}}<div class="err">{{.Error}}</div>{{end}}
    <form method="POST" action="/auth">
      <input type="password" name="pin" inputmode="numeric" placeholder="PIN" autofocus required />
      <button type="submit">Unlock</button>
    </form>
  </div>
</body>
</html>`))
