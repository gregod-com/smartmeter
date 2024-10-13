package main

import (
	"image"
	"image/jpeg"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func setCamera(parameters map[string]string) {
	req, err := http.NewRequest("GET", "http://"+config.CameraIP+"/resolution", nil)
	if err != nil {
		log.Print(err)
	}
	q := req.URL.Query()

	if parameters == nil {
		parameters = map[string]string{
			"sx":      "0",
			"sy":      "0",
			"ex":      "200",
			"ey":      "200",
			"offx":    config.CameraSettings.OffsetX,
			"offy":    config.CameraSettings.OffsetY,
			"tx":      config.CameraSettings.Width,
			"ty":      config.CameraSettings.Height,
			"ox":      config.CameraSettings.Width,
			"oy":      config.CameraSettings.Height,
			"scale":   "false",
			"binning": "true",
		}
	}
	for key, value := range parameters {
		q.Add(key, value)
	}

	req.URL.RawQuery = q.Encode()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error fetching data:", err)
		return
	}
	defer resp.Body.Close()
}

func controlCamera(parameters map[string]string) {
	req, err := http.NewRequest("GET", "http://"+config.CameraIP+"/control", nil)
	if err != nil {
		log.Print(err)
	}
	q := req.URL.Query()
	for key, value := range parameters {
		q.Add("var", key)
		q.Add("val", value)
	}

	req.URL.RawQuery = q.Encode()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error fetching data:", err)
		return
	}
	defer resp.Body.Close()
}

func initCameraSettings(ledStrength string) {
	setCamera(nil)
	controlCamera(map[string]string{"led_intensity": string(ledStrength)})                  // 220 = led intensity (0-255)
	controlCamera(map[string]string{"quality": config.CameraSettings.Quality})              // 4 = high quality
	controlCamera(map[string]string{"special_effect": config.CameraSettings.SpecialEffect}) // 1 = negative
	controlCamera(map[string]string{"hmirror": "1"})                                        // 1 = horizontal mirror
	controlCamera(map[string]string{"vflip": "1"})                                          // 1 = vertical flip
	controlCamera(map[string]string{"lenc": "1"})                                           // 1 = lens correction
	controlCamera(map[string]string{"gainceiling": config.CameraSettings.GainCeiling})      // 1 = gain ceiling (0-6)
	controlCamera(map[string]string{"wb_mode": config.CameraSettings.WbMode})               // 1 = white balance mode (0-4)
}

func fetchNewPhoto(ledStrength string) image.Image {
	initCameraSettings(ledStrength)

	resp, err := http.Get("http://" + config.CameraIP + "/capture")
	if err != nil {
		log.Println("Error fetching data:", err)
		return nil
	}
	defer resp.Body.Close()

	img, err := jpeg.Decode(resp.Body)
	if err != nil {
		log.Println("Error decoding image:", err)
	}

	return img
}
