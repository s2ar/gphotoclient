package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	gphotos "github.com/gphotosuploader/google-photos-api-client-go/v2"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	photoScopes = []string{
		"https://www.googleapis.com/auth/photoslibrary.appendonly",
		"https://www.googleapis.com/auth/photoslibrary.readonly",
		"https://www.googleapis.com/auth/photoslibrary.readonly.appcreateddata",
	}
	ignoredFile   = "ignored_file.txt"
	successedFile = "successed_file.txt"
	pattern       = "%s#~^^^~#%s"
	uploadedFiles = make(map[string]string)
	key           string
	path          string
	found         bool
	limit         int
	step          int
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func dirTree(ctx context.Context, client *gphotos.Client, out io.Writer, path string, uploadedFiles map[string]string, limit *int, step int) error {
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("dirTree: %w", err)
	}
	return recDirTree(ctx, client, out, pathAbs, 0, uploadedFiles, limit, step)
}

func recDirTree(ctx context.Context, client *gphotos.Client, out io.Writer, path string, lvl int, uploadedFiles map[string]string, limit *int, step int) error {

	var size int64
	var pathList []string
	var availableExt = []string{".png", ".JPG", ".jpg"}

	lvl++
	hPath, _ := os.Open(path)
	fInfo, _ := hPath.Stat()

	if fInfo.IsDir() {
		pathList, _ = hPath.Readdirnames(10000)
		sort.Strings(pathList)
	}

	for index := range pathList {

		absolutePath := filepath.Join(path, pathList[index])
		hPathFile, _ := os.Open(absolutePath)
		fInfoFile, _ := hPathFile.Stat()

		if !fInfoFile.IsDir() {
			size = fInfoFile.Size()
			if size > 0 {

				// проверяем расширение файла
				if isContains(availableExt, filepath.Ext(pathList[index])) {

					key = pathList[index] + "," + strconv.FormatInt(size, 10)
					_, found = uploadedFiles[key]
					if !found {
						// пользовательская логика
						// после успеха запись в мапу и сохранение в файл

						err := uploadMedia(ctx, client, absolutePath)
						if err == nil {
							fmt.Println(*limit, absolutePath)
							uploadedFiles[key] = absolutePath
							saveSuccessedToFile(key, absolutePath)
							//fmt.Fprintln(out, absolutePath)
							*limit++

							if *limit >= step {
								fmt.Println("Limit reached")
								os.Exit(0)
							}
						} else {
							err := fmt.Errorf(strconv.Itoa(*limit)+", "+absolutePath+", uploadMedia: %w", err)
							fmt.Println(err)
							return err
						}

					}
					//fmt.Println(uploadedFiles)

				} else {
					// фиксируем те которые не прошли
					saveIgnoredExtToFile(absolutePath)
				}
			}
		}
		//fmt.Fprintln(out, sepRow+pathList[index]+sizeStr)
		if fInfo.IsDir() {
			recDirTree(ctx, client, out, filepath.Join(path, pathList[index]), lvl, uploadedFiles, limit, step)
		}
	}

	return nil

}

func isContains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func saveIgnoredExtToFile(ignoredPath string) {

	f, err := os.OpenFile(ignoredFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	check(err)
	defer f.Close()

	datawriter := bufio.NewWriter(f)
	_, _ = datawriter.WriteString(ignoredPath + "\n")
	datawriter.Flush()
}

func saveSuccessedToFile(key string, successedPath string) {
	f, err := os.OpenFile(successedFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	check(err)
	defer f.Close()

	row := fmt.Sprintf(pattern, key, successedPath)
	datawriter := bufio.NewWriter(f)
	_, _ = datawriter.WriteString(row + "\n")
	datawriter.Flush()
}

func readSuccessedFromFile() (map[string]string, error) {
	lines := make(map[string]string)
	var s []string

	if _, err := os.Stat(successedFile); err == nil || os.IsExist(err) {
		file, err := ioutil.ReadFile(successedFile)
		if err != nil {
			return lines, err
		}
		buf := bytes.NewBuffer(file)
		for {
			line, err := buf.ReadString('\n')
			if len(line) == 0 {
				if err != nil {
					if err == io.EOF {
						break
					}
					return lines, err
				}
			}
			s = strings.Split(line, fmt.Sprintf(pattern, "", ""))
			if len(s) != 2 {
				continue
			}

			lines[s[0]] = s[1]
			if err != nil && err != io.EOF {
				return lines, err
			}
		}
	}
	return lines, nil
}

func uploadMedia(ctx context.Context, client *gphotos.Client, absolutePath string) error {
	item := gphotos.FileUploadItem(absolutePath)
	//_, err = client.AddMediaToAlbum(ctx, item, album)
	_, err := client.AddMediaToLibrary(ctx, item)

	return err
}

func main() {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, photoScopes...)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	httpClient := getClient(config)

	ctx := context.Background()

	// httpClient is an authenticated http.Client. See Authentication below.
	// You can customize the client using Options, see below.
	//client := gphotos.NewClient(client)
	//client, err := gphotos.NewClient(client)

	client, err := gphotos.NewClient(httpClient)

	if err != nil {
		log.Fatalf("New client not created. %v", err)
	}

	// логика сканирование

	flag.StringVar(&path, "path", path, "директория для сканирования")
	flag.IntVar(&step, "step", 10, "кол-во успешных отправок за один запуск")

	flag.Parse()

	if _, err := os.Stat(path); err == nil || os.IsExist(err) {
	} else {
		check(err)
	}

	if _, err := os.Stat(ignoredFile); err == nil || os.IsExist(err) {
		err := os.Remove(ignoredFile)
		check(err)
	}

	uploadedFiles, err := readSuccessedFromFile()
	check(err)

	out := os.Stdout

	// перед проходом по директории, восстановим мапу с файла
	err = dirTree(ctx, client, out, path, uploadedFiles, &limit, step)
	check(err)

	// create a Photos Album with the specified name.
	/*
		title := "FIRST"

		album, err := client.FindAlbum(ctx, title)
		if err != nil {
			album, err = client.CreateAlbum(ctx, title)
			if err != nil {
				log.Fatalf("222222. %v", err)
			}
		}*/

	// upload an specified file to the previous album.
	/*
		item := gphotos.FileUploadItem("/home/s2ar/Pictures/DSC00087.JPG")
		//_, err = client.AddMediaToAlbum(ctx, item, album)
		_, err = client.AddMediaToLibrary(ctx, item)
		if err != nil {
			log.Fatalf("333333. %v", err)
		}
	*/
}
