package main

import (
	"crypto/rand"
	"crypto/subtle"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

const appVersion = "1.0.0"

type serverConfig struct {
	targetPath  string
	isSingle    bool
	uploadDir   string
	allowUpload bool
	pin         string
}

func main() {
	port := flag.Int("port", 8080, "TCP port to listen on (0 for random free port)")
	allowUpload := flag.Bool("upload", false, "Allow uploading files from phone/browser to host")
	pinFlag := flag.String("pin", "", "Protect transfer with a PIN (use 'auto' for random 4-digit PIN)")
	noQR := flag.Bool("no-qr", false, "Disable terminal QR code output")
	showVersion := flag.Bool("version", false, "Print version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "r8-lanshare v%s — Fast Local Wi-Fi File Transfer with QR Code (R8Force)\n\n", appVersion)
		fmt.Fprintf(os.Stderr, "Usage:\n  %s [options] [file_or_directory]\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("r8-lanshare %s (R8Force)\n", appVersion)
		return
	}

	targetArg := "."
	if flag.NArg() > 0 {
		targetArg = flag.Arg(0)
	}

	absTarget, err := filepath.Abs(targetArg)
	if err != nil {
		log.Fatalf("Error resolving path: %v", err)
	}

	info, err := os.Stat(absTarget)
	if err != nil {
		log.Fatalf("Error accessing '%s': %v", targetArg, err)
	}

	cfg := &serverConfig{
		targetPath:  absTarget,
		isSingle:    !info.IsDir(),
		allowUpload: *allowUpload,
	}

	if info.IsDir() {
		cfg.uploadDir = absTarget
	} else {
		cfg.uploadDir = filepath.Dir(absTarget)
	}

	if *pinFlag == "auto" {
		cfg.pin = generatePin()
	} else {
		cfg.pin = strings.TrimSpace(*pinFlag)
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", *port, err)
	}
	defer ln.Close()

	actualPort := ln.Addr().(*net.TCPAddr).Port
	lanIP := detectLANIP()
	shareURL := fmt.Sprintf("http://%s:%d/", lanIP, actualPort)

	printBanner(cfg, shareURL, *noQR)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth", cfg.handleAuth)
	mux.HandleFunc("/upload", cfg.withAuth(cfg.handleUpload))
	mux.HandleFunc("/download", cfg.withAuth(cfg.handleSingleDownload))
	mux.HandleFunc("/", cfg.withAuth(cfg.handleBrowse))

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	if err := srv.Serve(ln); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}

func printBanner(cfg *serverConfig, shareURL string, noQR bool) {
	mode := "Directory"
	if cfg.isSingle {
		mode = "Single File"
	}

	fmt.Println("==================================================")
	fmt.Printf(" R8Force LANShare v%s\n", appVersion)
	fmt.Println("==================================================")
	fmt.Printf("  Mode      : %s\n", mode)
	fmt.Printf("  Target    : %s\n", cfg.targetPath)
	if cfg.allowUpload {
		fmt.Printf("  Uploads   : Enabled (saving to %s)\n", cfg.uploadDir)
	} else {
		fmt.Println("  Uploads   : Disabled (use -upload to enable)")
	}
	if cfg.pin != "" {
		fmt.Printf("  PIN Lock  : %s\n", cfg.pin)
	}
	fmt.Printf("  Open URL  : %s\n", shareURL)
	fmt.Println("--------------------------------------------------")

	if !noQR {
		qr, err := qrcode.New(shareURL, qrcode.Medium)
		if err == nil {
			fmt.Println("Scan this QR code with your phone (same Wi-Fi):")
			fmt.Print(qr.ToSmallString(false))
		}
	}
	fmt.Println("Press Ctrl+C to stop the server.")
	fmt.Println()
}

func (c *serverConfig) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c.pin == "" {
			next(w, r)
			return
		}
		cookie, err := r.Cookie("r8_pin")
		if err == nil && constantTimeEqual(cookie.Value, c.pin) {
			next(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_ = pinTmpl.Execute(w, PinPageData{Version: appVersion})
	}
}

func (c *serverConfig) handleAuth(w http.ResponseWriter, r *http.Request) {
	if c.pin == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	entered := strings.TrimSpace(r.FormValue("pin"))
	if !constantTimeEqual(entered, c.pin) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_ = pinTmpl.Execute(w, PinPageData{Version: appVersion, Error: "Incorrect PIN. Try again."})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "r8_pin",
		Value:    c.pin,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (c *serverConfig) handleSingleDownload(w http.ResponseWriter, r *http.Request) {
	if !c.isSingle {
		http.NotFound(w, r)
		return
	}
	name := filepath.Base(c.targetPath)
	fmt.Printf("  [DOWNLOAD] %s -> %s\n", name, r.RemoteAddr)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(name)))
	http.ServeFile(w, r, c.targetPath)
}

func (c *serverConfig) handleUpload(w http.ResponseWriter, r *http.Request) {
	if !c.allowUpload {
		http.Error(w, "Uploads are disabled. Restart with -upload flag.", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "Invalid multipart upload request.", http.StatusBadRequest)
		return
	}

	savedCount := 0
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "Upload interrupted.", http.StatusBadRequest)
			return
		}

		rawName := part.FileName()
		if rawName == "" {
			_ = part.Close()
			continue
		}

		safeName := filepath.Base(strings.ReplaceAll(rawName, "\\", "/"))
		if safeName == "" || safeName == "." || safeName == "/" {
			_ = part.Close()
			continue
		}

		destPath := uniqueFilePath(c.uploadDir, safeName)
		dst, err := os.Create(destPath)
		if err != nil {
			_ = part.Close()
			http.Error(w, "Could not save file on host.", http.StatusInternalServerError)
			return
		}

		written, copyErr := io.Copy(dst, part)
		closeErr := dst.Close()
		_ = part.Close()

		if copyErr != nil || closeErr != nil {
			_ = os.Remove(destPath)
			http.Error(w, "Error writing uploaded file.", http.StatusInternalServerError)
			return
		}

		fmt.Printf("  [UPLOAD]   %s (%s) <- %s\n", filepath.Base(destPath), formatBytes(written), r.RemoteAddr)
		savedCount++
	}

	msg := fmt.Sprintf("Uploaded %d file(s) successfully.", savedCount)
	http.Redirect(w, r, "/?msg="+url.QueryEscape(msg), http.StatusSeeOther)
}

func (c *serverConfig) handleBrowse(w http.ResponseWriter, r *http.Request) {
	if c.isSingle {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		info, err := os.Stat(c.targetPath)
		if err != nil {
			http.Error(w, "Shared file is no longer available.", http.StatusNotFound)
			return
		}
		data := PageData{
			Version:     appVersion,
			SingleFile:  true,
			AllowUpload: c.allowUpload,
			Notice:      r.URL.Query().Get("msg"),
			SingleItem: FileItem{
				Name:    info.Name(),
				Href:    "/download",
				SizeStr: formatBytes(info.Size()),
			},
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = pageTmpl.Execute(w, data)
		return
	}

	relClean := filepath.Clean("/" + r.URL.Path)
	relTrim := strings.TrimPrefix(relClean, "/")
	fullPath := filepath.Join(c.targetPath, filepath.FromSlash(relTrim))

	if !isSubPath(c.targetPath, fullPath) {
		http.Error(w, "Forbidden path.", http.StatusForbidden)
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if !info.IsDir() {
		name := info.Name()
		fmt.Printf("  [DOWNLOAD] %s -> %s\n", name, r.RemoteAddr)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(name)))
		http.ServeFile(w, r, fullPath)
		return
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		http.Error(w, "Cannot read directory.", http.StatusInternalServerError)
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	items := make([]FileItem, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		sizeStr := "DIR"
		if !entry.IsDir() {
			if fi, statErr := entry.Info(); statErr == nil {
				sizeStr = formatBytes(fi.Size())
			} else {
				sizeStr = "-"
			}
		}
		items = append(items, FileItem{
			Name:    name,
			Href:    encodeURLPath(relTrim, name),
			SizeStr: sizeStr,
			IsDir:   entry.IsDir(),
		})
	}

	hasParent := relTrim != ""
	parentHref := "/"
	if hasParent {
		parentRel := filepath.ToSlash(filepath.Dir(relTrim))
		if parentRel != "." && parentRel != "/" {
			parentHref = encodeURLPath(parentRel)
		}
	}

	displayPath := "/" + relTrim
	data := PageData{
		Version:     appVersion,
		Title:       displayPath,
		AllowUpload: c.allowUpload,
		Items:       items,
		Notice:      r.URL.Query().Get("msg"),
		HasParent:   hasParent,
		ParentHref:  parentHref,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pageTmpl.Execute(w, data)
}

func detectLANIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok && addr.IP.To4() != nil {
			return addr.IP.String()
		}
	}

	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, a := range addrs {
			if ipNet, ok := a.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ip4 := ipNet.IP.To4(); ip4 != nil {
					return ip4.String()
				}
			}
		}
	}
	return "127.0.0.1"
}

func generatePin() string {
	n, err := rand.Int(rand.Reader, big.NewInt(9000))
	if err != nil {
		return "8080"
	}
	return fmt.Sprintf("%04d", n.Int64()+1000)
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func isSubPath(base, target string) bool {
	if target == base {
		return true
	}
	prefix := base
	if !strings.HasSuffix(prefix, string(os.PathSeparator)) {
		prefix += string(os.PathSeparator)
	}
	return strings.HasPrefix(target, prefix)
}

func uniqueFilePath(dir, name string) string {
	candidate := filepath.Join(dir, name)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; i < 10000; i++ {
		next := filepath.Join(dir, fmt.Sprintf("%s_%d%s", stem, i, ext))
		if _, err := os.Stat(next); os.IsNotExist(err) {
			return next
		}
	}
	return filepath.Join(dir, fmt.Sprintf("%d_%s", time.Now().Unix(), name))
}

func encodeURLPath(parts ...string) string {
	var segs []string
	for _, p := range parts {
		for _, s := range strings.Split(p, "/") {
			if s == "" || s == "." {
				continue
			}
			segs = append(segs, url.PathEscape(s))
		}
	}
	return "/" + strings.Join(segs, "/")
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
