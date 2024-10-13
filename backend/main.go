package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
)

var (
	fetchedData string
	config      Config
)

type Config struct {
	OpenaiAPIKey         string `yaml:"openai_api_key"`
	Port                 string `yaml:"port"`
	UserPrompt           string `yaml:"user_prompt"`
	BrightnessThresholds struct {
		Min int `yaml:"min"`
		Max int `yaml:"max"`
	} `yaml:"brightness_thresholds"`
	ChatModel      string `yaml:"chat_model"`
	CameraIP       string `yaml:"camera_ip"`
	DatabasePath   string `yaml:"database_path"`
	CameraSettings struct {
		LedIntensity  string `yaml:"led_intensity"`
		Quality       string `yaml:"quality"`
		SpecialEffect string `yaml:"special_effect"`
		GainCeiling   string `yaml:"gain_ceiling"`
		WbMode        string `yaml:"wb_mode"`
		OffsetX       string `yaml:"offset_x"`
		OffsetY       string `yaml:"offset_y"`
		Width         string `yaml:"width"`
		Height        string `yaml:"height"`
	} `yaml:"camera_settings"`
}

func init() {
	// read config.yaml from local disk and unmarshal it into Config struct
	err := readConfig("config.yaml", &config)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}
	initDB()
}

func readConfig(filename string, config *Config) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return yaml.NewDecoder(file).Decode(config)
}

// Function to calculate the average brightness of an image
func calculateAverageBrightness(img image.Image) float64 {
	bounds := img.Bounds()
	var totalBrightness float64
	var totalPixels int

	// Loop through each pixel
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()

			// Convert the pixel to grayscale
			brightness := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
			totalBrightness += brightness
			totalPixels++
		}
	}

	// Return the average brightness
	return totalBrightness / float64(totalPixels)
}

func fetchNewDataHandler(w http.ResponseWriter, r *http.Request) {
	img := fetchNewPhoto(config.CameraSettings.LedIntensity)

	// while calculateAverageBrightness returns the average brightness of below 50, fetch a new photo
	for brightness := calculateAverageBrightness(img); brightness < float64(config.BrightnessThresholds.Min) || brightness > float64(config.BrightnessThresholds.Max); brightness = calculateAverageBrightness(img) {
		log.Debug("Average brightness:", brightness)
		if brightness < float64(config.BrightnessThresholds.Min) {
			img = fetchNewPhoto("50")
		} else {
			img = fetchNewPhoto(config.CameraSettings.LedIntensity)
		}
	}

	buf := new(bytes.Buffer)

	err := jpeg.Encode(buf, img, nil)
	if err != nil {
		log.Warn("Error decoding image:", err)
	}

	// body contains a jpeg image, convert it to base64
	fetchedData = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	ocr_data, err := ocrDecodeImage(fetchedData)
	if err != nil {
		log.Warn("Error decoding image:", err)
	}

	var gasMeterReading GasMeterReading
	gasMeterReading.Date = time.Now()
	gasMeterReading.Brightness = int(calculateAverageBrightness(img))
	gasMeterReading.ImageData = fetchedData
	gasMeterReading.OCRData = ocr_data
	gasMeterReading.Measurement, err = strconv.ParseFloat(ocr_data, 64)
	if err != nil {
		log.Warn("Error parsing OCR data:", err)
	}

	// first check if there is a delta to the previous reading and if the value is sane
	// func getPaginated(object interface{}, start, end int, sort, order string) error {
	previousReading := GasMeterReading{}
	err = getPaginated(&previousReading, 0, 1, "date", "DESC")
	if err != nil {
		log.Warnf("Failed to get previous GasMeterReading: %v", err)
	}

	if previousReading.Measurement == 0 {
		log.Info("Previous reading measurement not set. Converting ocr_data to float64.")
		previousReading.Measurement, err = strconv.ParseFloat(previousReading.OCRData, 64)
		if err != nil {
			log.Warn("Error parsing previous reading OCR data:", err)
		}
	}

	switch {
	case gasMeterReading.Measurement < previousReading.Measurement:
		log.Info("New reading is lower than previous reading. Previous:", previousReading.Measurement, "New:", gasMeterReading.Measurement)
		log.Info("Dropping new reading.")
		return
	case gasMeterReading.Measurement == previousReading.Measurement:
		log.Debug("New reading is the same as previous reading. Previous:", previousReading.Measurement, "New:", gasMeterReading.Measurement)
		log.Debug("Dropping new reading.")
		return
	case gasMeterReading.Measurement-previousReading.Measurement > 5:
		log.Warnf("New reading is more than 5 m3 higher than previous reading. Previous: %v, New: %v", previousReading.Measurement, gasMeterReading.Measurement)
		log.Warn("Dropping new reading, because unlikely to be correct.")
		return
	default:
		log.Debug("Previous:", previousReading.Measurement, "New:", gasMeterReading.Measurement)
		// Save the object
		err = createGasMeterReading(&gasMeterReading)
		if err != nil {
			log.Warnf("Failed to save GasMeterReading: %v", err)
		}
	}
}

func getGasMeterReading(w http.ResponseWriter, r *http.Request) {
	// get id from request path e.g. `/gasmeterreadings/1`
	id := strings.TrimPrefix(r.URL.Path, "/gasmeterreadings/")
	reading := GasMeterReading{}
	err := read(&reading, id)
	if err != nil {
		log.Errorf("Failed to get GasMeterReading. Err: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK) // 200 - OK (data is ready)
	json.NewEncoder(w).Encode(reading)
}

func getGasMeterReadings(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	start := r.URL.Query().Get("_start")
	end := r.URL.Query().Get("_end")
	order := r.URL.Query().Get("_order")
	sort := r.URL.Query().Get("_sort")

	// Convert start and end to integers
	startInt, err := strconv.Atoi(start)
	if err != nil {
		startInt = 0
	}
	endInt, err := strconv.Atoi(end)
	if err != nil {
		endInt = 10
	}
	if sort == "" {
		sort = "id"
	}
	if order == "" {
		order = "ASC"
	}

	// Get paginated entries
	paginatedEntries := []GasMeterReading{}
	err = getPaginated(&paginatedEntries, startInt, endInt, sort, order)
	if err != nil {
		log.Warnf("Failed to get paginated GasMeterReading: %v", err)
	}
	count, err := count(&GasMeterReading{})
	if err != nil {
		log.Warn("Failed to get count of GasMeterReading")
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.FormatInt(count, 10))

	w.WriteHeader(http.StatusOK) // 200 - OK (data is ready)
	json.NewEncoder(w).Encode(paginatedEntries)
}

func removeGasMeterReading(w http.ResponseWriter, r *http.Request) {
	// get id from request path e.g. `/delete/1`
	id := strings.TrimPrefix(r.URL.Path, "/gasmeterreadings/")
	err := deleteGasMeterReading(&GasMeterReading{}, id)
	if err != nil {
		log.Warnf("Failed to delete all GasMeterReading: %v", err)
	}
	log.Info("Deleted entry with ID:", id)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 204 No Content
	json.NewEncoder(w).Encode(id)
}

func addGasMeterReading(w http.ResponseWriter, r *http.Request) {
	var gasMeterReading GasMeterReading
	err := json.NewDecoder(r.Body).Decode(&gasMeterReading)
	if err != nil {
		log.Warnf("Failed to add GasMeterReading: %v", err)
	}
	log.Infof("Adding entry: %v", gasMeterReading)
	err = createGasMeterReading(&gasMeterReading)
	w.WriteHeader(http.StatusNotImplemented)
}

func putGasMeterReading(w http.ResponseWriter, r *http.Request) {
	// get id from request path e.g. `/gasmeterreadings/1`
	id := strings.TrimPrefix(r.URL.Path, "/gasmeterreadings/")
	log.Warnf("Updating not implemented yet. ID: %s", id)
	w.WriteHeader(http.StatusNotImplemented)
}

func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func main() {
	r := mux.NewRouter()

	// Handle home page and fetch actions
	r.HandleFunc("/gasmeterreadings/{id}", getGasMeterReading).Methods("GET")
	r.HandleFunc("/gasmeterreadings/{id}", putGasMeterReading).Methods("PUT")
	r.HandleFunc("/gasmeterreadings/{id}", removeGasMeterReading).Methods("DELETE")
	r.HandleFunc("/gasmeterreadings", getGasMeterReadings).Methods("GET")
	r.HandleFunc("/gasmeterreadings", addGasMeterReading).Methods("POST")
	r.HandleFunc("/fetchwithnew", fetchNewDataHandler)

	// serve all files from folder `test-admin/dist` under root or request
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("dist")))

	// Initialize CORS settings
	headers := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	methods := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "DELETE"})
	origins := handlers.AllowedOrigins([]string{"*"})                    // Allow all origins for testing
	exposedHeaders := handlers.ExposedHeaders([]string{"X-Total-Count"}) // Expose X-Total-Count
	optionsStatusCode := handlers.OptionStatusCode(http.StatusNoContent)

	// Combine CORS and Log middleware
	corsHandler := handlers.CORS(origins, headers, methods, optionsStatusCode, exposedHeaders)(r)

	log.Infof("Server started on :%s", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, Log(corsHandler)))
}
