package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

func IsOwnGame(product_id string) bool {
	data, err := client.GetUserGames()
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range data {
		if v.ProductID == product_id {
			return true
		}
	}

	return false
}

func CheckIfInstalled(product_id string) bool {
	for _, v := range config.Games {
		if product_id == v.ProductID {
			return true
		}
	}

	return false
}

func GetGameConfigIndex(product_id string) int {
	for i, v := range config.Games {
		if product_id == v.ProductID {
			return i
		}
	}

	return 0
}

func GetGameConfig(product_id string) *Game {
	for _, v := range config.Games {
		if product_id == v.ProductID {
			return &v
		}
	}

	return nil
}

func LaunchGame(product_id string) {
	if err := CheckAccessToken(); err != nil {
		log.Fatal(err)
	}

	if CheckIfInstalled(product_id) == false {
		log.Fatal(product_id, " is not installed. Install it first with -install")
	}

	log.Printf("Getting game details...")
	data, err := client.LaunchGame(product_id, "GCL")
	if err != nil {
		log.Fatal(err)
	}

	c := GetGameConfig(product_id)
	cmd := path.Join(c.Location, data.ProductID, data.ExecFilename)
	args := strings.Split(data.ExecuteArgs, " ")

	log.Printf("Product: %s", data.ProductID)
	log.Printf("Version: %s (local: %s)", data.LatestVersion, c.Version)
	log.Printf("Args: %s", data.ExecuteArgs)
	log.Printf("Executable: %s", cmd)

	log.Printf("Launching...")

	_cmd := exec.Command(cmd, args...)
	_cmd.Start()
	_cmd.Wait()

	log.Print("Process exited")
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
		Timeout: 2 * time.Minute,
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

	log.Printf("Starting download with 5 jobs")

	for page := range tp.Data.TotalPages {
		var fl dmmgo.DMMResponse[FileListResp]

		resp, err := httpc.Get("https://apidgp-gameplayer.games.dmm.com" + file_list_url+ "?page=" + strconv.Itoa(page))

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

		for range 5 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range ch {
					err := install_worker(ctx, httpc, j, dest_dir, sign, gamecfg)
					if err != nil {
						log.Print(err)
					}
				}
			}()
		}

		go func() {
			defer close(ch)
			for _, f := range fl.Data.FileList {
				select {
					case <-ctx.Done():
						return
					case ch <- f:
				}
			}
		}()

		wg.Wait()
	}
}

func InstallGame(ctx context.Context, product_id string, dest_dir string) {
	if err := CheckAccessToken(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Determining ownership for %s...", product_id)
	if ok := IsOwnGame(product_id); !ok {
		log.Fatalf("You do not own %s. You need to add it to your library/buy it first from the official client.", product_id)
	}

	log.Printf("Requesting game information for %s...", product_id)
	data, err := client.GetInstallData(product_id, "GCL")
	if err != nil {
		log.Fatal(err)
	}

	fold := path.Join(dest_dir, product_id)
	log.Print("Destination directory:", fold)

	if CheckIfInstalled(product_id) != true {
		config.Games = append(config.Games, Game{
			ProductID: product_id,
			Location:  dest_dir,
			Version:   data.LatestVersion,
		})
		WriteConfig(config)
	} else {
		idx := GetGameConfigIndex(product_id)
		if config.Games[idx].Location != dest_dir {
			config.Games[idx].Location = dest_dir
			WriteConfig(config)
		}
	}

	gconf := GetGameConfig(product_id)
	DoGameDownload(ctx, data.FileListURL, fold, data.Sign, *gconf)

	log.Print("Finished downloading")
}
