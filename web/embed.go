package web

import "embed"

// Assets contains the standalone WeClaw administration interface.
//
//go:embed index.html styles.css app.js
var Assets embed.FS
