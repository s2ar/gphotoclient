package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	gphotos "github.com/gphotosuploader/google-photos-api-client-go/v2"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var photoScopes = []string{
	"https://www.googleapis.com/auth/photoslibrary.appendonly",
	"https://www.googleapis.com/auth/photoslibrary.readonly",
	"https://www.googleapis.com/auth/photoslibrary.readonly.appcreateddata",
}

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
		log.Fatalf("111111. %v", err)
	}

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
	item := gphotos.FileUploadItem("/home/s2ar/Pictures/DSC00087.JPG")
	//_, err = client.AddMediaToAlbum(ctx, item, album)
	_, err = client.AddMediaToLibrary(ctx, item)
	if err != nil {
		log.Fatalf("333333. %v", err)
	}
}
