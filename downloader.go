package main

import (
	"context"
	"net/http"
	"path"
	"os"
	"path/filepath"
	"log"
	"sync"
	"errors"
	"bytes"
	"fmt"
	"encoding/json"
	"strconv"
	"io"
	"time"

	dmmgo "codeberg.org/kita/dmm-go"
)

type TotalPagesResp struct {
	TotalPages int `json:"total_pages"`
}

type File struct {
	LocalPath   string `json:"local_path"`
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	Hash        string `json:"hash"`
	Protected   bool   `json:"protected_flg"`
	ForceDelete bool   `json:"force_delete"`
	CheckHash   bool   `json:"check_hash"`
	Timestamp   int64  `json:"timestamp"`
}

type FileListResp struct {
	Domain   string `json:"domain"`
	FileList []File `json:"file_list"`
	Page     int    `json:"page"`
}


func install_worker(ctx context.Context, client *http.Client, file File, dest_dir string, policy string, gamecfg Game) error {
	dest := path.Join(dest_dir, file.LocalPath)
	dest_tmp := dest + ".tmp"

	_, err := os.Stat(dest)

	if err == nil {
		for _, v := range gamecfg.IgnoreFileList {
			if filepath.Base(file.LocalPath) == v {
				log.Printf("Ignoring file %s because it's in the ignore list", file.LocalPath)
				return nil
			}
		}

		log.Printf("Checking md5 for %s", file.LocalPath)
		h, err := fileMD5Sum(dest)
		if err != nil {
			return err
		}

		if h != file.Hash {
			log.Printf("Hash for %s does not match, redownloading it", file.LocalPath)
		} else {
			log.Printf("Skip %s: file exists", file.LocalPath)
			return nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	u := "https://cdn-gameplayer.games.dmm.com/" + file.Path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, bytes.NewBuffer([]byte("")))
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", policy)

	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected status code %d trying to retrieve %s", resp.StatusCode, u)
	}

	log.Print("Downloading ", file.LocalPath)

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	out, err := os.Create(dest_tmp)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		_ = os.Remove(dest_tmp)
		return err
	}

	if err := out.Close(); err != nil {
		_ = os.Remove(dest_tmp)
		return err
	}

	log.Printf("Checking md5 for %s", file.LocalPath)
	h, err := fileMD5Sum(dest_tmp)
	if err != nil {
		return err
	}

	if h != file.Hash {
		return fmt.Errorf("File hash for %s do not match", file.LocalPath)
	}

	log.Print("Downloaded ", file.LocalPath)
	return os.Rename(dest_tmp, dest)
}

func DoGameDownload(ctx context.Context, file_list_url string, dest_dir string, sign string, gamecfg Game) {
	ch := make(chan File)
	var wg sync.WaitGroup

	var tp dmmgo.DMMResponse[TotalPagesResp]

	httpc := &http.Client{
		Timeout: 10 * time.Minute,
	}

	resp, err := httpc.Get("https://apidgp-gameplayer.games.dmm.com" + file_list_url + "/totalpages")

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	if err := json.Unmarshal(content, &tp); err != nil {
		log.Fatal(err)
	}

	var files []File

	log.Printf("Starting download with 5 jobs")

	for page := range tp.Data.TotalPages {
		var fl dmmgo.DMMResponse[FileListResp]

		resp, err := httpc.Get("https://apidgp-gameplayer.games.dmm.com" + file_list_url + "?page=" + strconv.Itoa(page+1))

		if err != nil {
			log.Fatal(err)
		}

		defer resp.Body.Close()

		content, err := io.ReadAll(resp.Body)

		if err != nil {
			log.Fatal(err)
		}

		if err := json.Unmarshal(content, &fl); err != nil {
			log.Fatal(err)
		}

		for _, f := range fl.Data.FileList {
			files = append(files, f)
		}
	}

	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range ch {
				err := install_worker(ctx, httpc, j, dest_dir, sign, gamecfg)
				if err != nil {
					log.Printf("%s: %s", j.LocalPath, err)
				}
			}
		}()
	}

	go func() {
		defer close(ch)
		for _, f := range files {
			select {
			case <-ctx.Done():
				return
			case ch <- f:
			}
		}
	}()

	wg.Wait()

}
