package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type ImgurProvider struct {
	album    string
	interval int
	client   http.Client
}

func buildImgurURL(endpoint string) string {
	return fmt.Sprintf("https://api.imgur.com%v", endpoint)
}

func (ip *ImgurProvider) imgurRequest(request *http.Request) (interface{}, error) {
	request.Header.Add("Authorization", "Client-ID 61128aab04600a9")
	response, err := ip.client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != 200 {
		logger.Infof("Something went wrong... %v => %v %v", response.Status, request.Method, request.URL)
	}
	defer response.Body.Close()
	bodyData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Infof("Failed to read body for %v %v", request.Method, request.URL)
		return nil, err
	}

	var value interface{}
	err = json.Unmarshal(bodyData, &value)
	if err != nil {
		logger.Infof("Unmarshalling failed... '%v'", bodyData)
		return nil, err
	}
	return value, err
}

func (ip *ImgurProvider) imgurGet(endpoint string) (interface{}, error) {
	request, err := http.NewRequest("GET", buildImgurURL(endpoint), nil)
	if err != nil {
		logger.Infof("Creating request failed")
		return nil, err
	}

	return ip.imgurRequest(request)
}

func (ip *ImgurProvider) imgurPost(endpoint string, body interface{}) (interface{}, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		logger.Infof("Failed to marshall")
		return nil, err
	}

	jsonReader := bytes.NewReader(jsonBody)
	request, err := http.NewRequest("POST", buildImgurURL(endpoint), jsonReader)
	if err != nil {
		logger.Infof("Creating request failed")
		return nil, err
	}

	return ip.imgurRequest(request)
}

func (ip *ImgurProvider) imgurPhotosFromAlbum(album string) ([]string, error) {
	albumEndpoint := fmt.Sprintf("/3/gallery/album/%v", album)
	result, err := ip.imgurGet(albumEndpoint)
	if err != nil {
		return nil, err
	}
	data := result.(map[string]interface{})["data"]
	imagesUrls := make([]string, 0)
	albumImages := data.(map[string]interface{})["images"]
	for _, imageI := range albumImages.([]interface{}) {
		image := imageI.(map[string]interface{})
		imageUrl := image["link"].(string)
		imagesUrls = append(imagesUrls, imageUrl)
	}
	return imagesUrls, nil
}

func (ip *ImgurProvider) getPhotos() ([]string, error) {
	return ip.imgurPhotosFromAlbum(ip.album)
}

func (ip *ImgurProvider) getName() string {
	return fmt.Sprintf("imgur-%v", ip.album)
}

func (ip *ImgurProvider) setStorageLocation(location string) {
}

func (ip *ImgurProvider) run(photoProvider *PhotoProvider) {
	var pp PhotoProvider = ip

	if photoProvider == nil {
		photoProvider = &pp
	}

	for {
		if photos, err := (*photoProvider).getPhotos(); err == nil {
			logger.Debugf("Got %v photos", len(photos))
		} else {
			logger.Infof("Failed to get photos from %v. %v", ip.album, err)
		}
		time.Sleep(time.Duration(ip.interval))
	}
}

func GetImgurPhotoProvider(config map[string]interface{}) PhotoProvider {
	album, ok := config["album"].(string)
	if !ok {
		logger.Errorf("path config parameter is not a string as expected")
		return nil
	}

	interval, ok := config["poll_interval"].(int)
	if !ok {
		interval = 1000
	}
	interval *= 1000000000

	var pl PhotoProvider = &PhotoDownloader{
		backend: &ImgurProvider{
			album:    album,
			interval: interval,
			client:   http.Client{},
		},
	}

	return pl
}

func init() {
	RegisterProvider("imgur", GetImgurPhotoProvider)
}
