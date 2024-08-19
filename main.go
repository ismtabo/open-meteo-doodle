package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/geo/s1"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Latitude   float64 `mapstructure:"latitude"`
	Longitude  float64 `mapstructure:"longitude"`
	OutputFile string  `mapstructure:"file"`
	Verbose    bool
}

type TimeLocation = time.Location

func init() {
	flag.StringP("config", "c", "", "Config file")
	flag.Float64("latitude", 0.0, "Latitude")
	flag.Float64("longitude", 0.0, "Longitude")
	flag.String("file", "", "Output file")
	flag.BoolP("verbose", "v", false, "Verbose mode")
	flag.ErrHelp = nil
	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(os.Stderr, "%s [options]\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "Options:")
	flag.PrintDefaults()
	os.Exit(0)
}

func main() {
	flag.Parse()
	if err := viper.BindPFlags(flag.CommandLine); err != nil {
		log.Fatal("Error: failed binding flags", err)
	}
	configFilePath := viper.GetString("config")
	if configFilePath != "" {
		viper.SetConfigFile(configFilePath)
		if err := viper.ReadInConfig(); err != nil {
			log.Fatal("Error: failed reading file", err)
		}
	}
	cfg := Config{}
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatal("Error: failed un-marshaling config", err)
	}
	if !cfg.Verbose {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))
		slog.SetDefault(logger)
	}
	log.Default().SetOutput(os.Stdout)
	if cfg.OutputFile == "" {
		log.Fatal("Error: missing output file option")
	}
	location := Location{
		Latitude:  cfg.Latitude,
		Longitude: cfg.Longitude,
	}
	weatherData, err := getOpenWeatherCurrentWeather(location)
	if err != nil {
		log.Fatal("Error: failed retrieving weather data", err)
	}
	city, err := getLatitudeLongitudeCity(location)
	if err != nil {
		log.Fatal("Error: failed retrieving reverse geocode data", err)
	}
	if err := drawImage(cfg, weatherData, city); err != nil {
		log.Fatal("Error: failed writing image", err)
	}
}

type OpenWeatherData struct {
	Timezone string `json:"timezone"`
	Current  struct {
		Time                     string  `json:"time"`
		Temperature2M            float64 `json:"temperature_2m"`
		RelativeHumidity         float64 `json:"relative_humidity_2m"`
		ApparentTemperature      float64 `json:"apparent_temperature"`
		IsDay                    int     `json:"is_day"`
		PrecipitationProbability float64 `json:"precipitation_probability"`
		WeatherCode              int     `json:"weather_code"`
		WindSpeed_10m            float64 `json:"wind_speed_10m"`
		WindDirection_10m        float64 `json:"wind_direction_10m"`
	} `json:"current"`
}

type WeatherData struct {
	Time                     time.Time
	Temperature              float64
	RelativeHumidity         float64
	ApparentTemperature      float64
	IsDay                    bool
	PrecipitationProbability float64
	WeatherCode              int
	WindSpeed                float64
	WindDirection            float64
}

type Location struct {
	Latitude  float64
	Longitude float64
}

func getOpenWeatherCurrentWeather(location Location) (WeatherData, error) {
	openWeatherURL := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current=temperature_2m,relative_humidity_2m,apparent_temperature,is_day,precipitation_probability,weather_code,wind_speed_10m,wind_direction_10m&timezone=Europe%%2FMadrid&forecast_days=1&forecast_hours=24", location.Latitude, location.Longitude)
	res, err := http.Get(openWeatherURL)
	if err != nil {
		return WeatherData{}, fmt.Errorf("error requesting weather info: %w", err)
	}
	buff := &bytes.Buffer{}
	io.Copy(buff, res.Body)
	slog.Info("weather response body", slog.String("body", buff.String()))
	data := OpenWeatherData{}
	if err := json.NewDecoder(buff).Decode(&data); err != nil {
		return WeatherData{}, fmt.Errorf("error decoding weather response: %w", err)
	}
	loc, err := time.LoadLocation(data.Timezone)
	if err != nil {
		return WeatherData{}, fmt.Errorf("error loading weather timezone: %w", err)
	}
	t, err := time.ParseInLocation("2006-01-02T15:04", data.Current.Time, loc)
	if err != nil {
		return WeatherData{}, fmt.Errorf("error decoding weather time: %w", err)
	}
	slog.Info("weather data", slog.Any("data", data))
	return WeatherData{
			Time:                     t,
			Temperature:              data.Current.Temperature2M,
			RelativeHumidity:         data.Current.RelativeHumidity,
			ApparentTemperature:      data.Current.ApparentTemperature,
			IsDay:                    data.Current.IsDay == 1,
			PrecipitationProbability: data.Current.PrecipitationProbability,
			WeatherCode:              data.Current.WeatherCode,
			WindSpeed:                data.Current.WindSpeed_10m,
			WindDirection:            data.Current.WindDirection_10m,
		},
		nil
}

type NominatimReverseGeocodeData struct {
	Address struct {
		City string `json:"city"`
	} `json:"address"`
}

type City string

func getLatitudeLongitudeCity(location Location) (City, error) {
	reverseGeocodingURL := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?format=json&lat=%f&lon=%f", location.Latitude, location.Longitude)
	slog.Info("reverse geocode url", slog.String("url", reverseGeocodingURL))
	req, err := http.NewRequest(http.MethodGet, reverseGeocodingURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating reverse geocode request: %w", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error requesting reverse geocode info: %w", err)
	}
	buff := &bytes.Buffer{}
	io.Copy(buff, res.Body)
	slog.Info("reverse geocode response body", slog.String("body", buff.String()))
	data := NominatimReverseGeocodeData{}
	if err := json.NewDecoder(buff).Decode(&data); err != nil {
		return "", fmt.Errorf("error decoding reverse geocode response: %w", err)
	}
	return City(data.Address.City), nil
}

func degreesToCardinal(degrees float64) string {
	switch {
	case degrees >= 337.5 || degrees < 22.5:
		return "N"
	case degrees < 45:
		return "NNE"
	case degrees < 67.5:
		return "NE"
	case degrees < 90:
		return "ENE"
	case degrees < 112.5:
		return "E"
	case degrees < 135:
		return "ESE"
	case degrees < 157.5:
		return "SE"
	case degrees < 180:
		return "SSE"
	case degrees < 202.5:
		return "S"
	case degrees < 225:
		return "SSW"
	case degrees < 247.5:
		return "SW"
	case degrees < 270:
		return "WSW"
	case degrees < 292.5:
		return "W"
	case degrees < 315:
		return "WNW"
	case degrees < 337.5:
		return "NW"
	default:
		return "Unknown"
	}
}

const SIZE = 300
const UBUNTU_FONT_PATH = "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
const TERMUX_FONT_PATH = "/data/data/com.termux/files/usr/share/fonts/TTF/DejaVuSans.ttf"

var fonts = map[string]string{
	"linux":   UBUNTU_FONT_PATH,
	"android": TERMUX_FONT_PATH,
}

//go:embed emojis/light/*.png
var lightEmojis embed.FS

//go:embed emojis/dark/*.png
var darkEmojis embed.FS

const UNKNOWN_EMOJI_FILE = "unknown.png"

func getWMOEmojiImage(isLight bool, code int) (image.Image, error) {
	folder := lightEmojis
	folderName := "emojis/light"
	if !isLight {
		folder = darkEmojis
		folderName = "emojis/dark"
	}
	entries, err := folder.ReadDir(folderName)
	if err != nil {
		return nil, fmt.Errorf("error opening emojis folder: %w", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), fmt.Sprintf("%d", code)) {
			emojiFile := path.Join(folderName, entry.Name())
			data, err := folder.Open(emojiFile)
			if err != nil {
				return nil, fmt.Errorf("error opening file '%s': %w", emojiFile, err)
			}
			img, err := png.Decode(data)
			if err != nil {
				return nil, fmt.Errorf("error opening png file '%s': %s", emojiFile, err)
			}
			return img, nil
		}
	}
	emojiFile := path.Join(folderName, UNKNOWN_EMOJI_FILE)
	data, err := folder.Open(emojiFile)
	if err != nil {
		return nil, fmt.Errorf("error opening file '%s': %w", emojiFile, err)
	}
	img, err := png.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("error opening png file '%s': %s", emojiFile, err)
	}
	return img, nil
}

var gray20 = color.RGBA{
	R: 51,
	G: 51,
	B: 51,
	A: 255,
}

var gray80 = color.RGBA{
	R: 204,
	G: 204,
	B: 204,
	A: 255,
}

func drawImage(cfg Config, weatherData WeatherData, location City) error {
	uname := runtime.GOOS
	fontPath, ok := fonts[uname]
	if !ok {
		return fmt.Errorf("unknown platform %s", uname)
	}
	w := SIZE
	h := SIZE
	dc := gg.NewContext(w, h)
	fgColor := gray80
	if weatherData.IsDay {
		fgColor = gray20
	}
	bgColor := gray20
	if weatherData.IsDay {
		bgColor = gray80
	}
	fs := float64(h) * 0.10
	dc.SetColor(bgColor)
	dc.Clear()
	dc.LoadFontFace(fontPath, fs)
	dc.SetColor(fgColor)
	dc.DrawStringAnchored("Temp:", 10, 10, 0, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%g ºC", weatherData.Temperature), 290, 10, 1, 1)
	dc.DrawStringAnchored("Humd:", 10, 40, 0, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%g %%", weatherData.RelativeHumidity), 290, 40, 1, 1)
	dc.DrawStringAnchored("Prep:", 10, 70, 0, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%g %%", weatherData.PrecipitationProbability), 290, 70, 1, 1)
	dc.DrawStringAnchored("Wind:", 10, 100, 0, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%g km/h", weatherData.WindSpeed), 290, 100, 1, 1)
	dc.DrawStringAnchored(degreesToCardinal(weatherData.WindDirection), 290, 130, 1, 1)
	emoji, err := getWMOEmojiImage(weatherData.IsDay, weatherData.WeatherCode)
	if err != nil {
		return fmt.Errorf("error getting emoji for isDay=%t code=%d: %w", weatherData.IsDay, weatherData.WeatherCode, err)
	}
	dc.DrawImageAnchored(emoji, 120, 150, 0.5, 0.20)
	smallFS := fs * 0.3
	dc.LoadFontFace(fontPath, smallFS)
	dc.Push()
	dc.Translate(265, 190)
	dc.DrawCircle(0, 0, 20)
	dc.DrawStringAnchored("N", 0, -24, 0.5, 0)
	dc.DrawLine(0, -20, 0, -18)
	dc.DrawStringAnchored("W", -24, 0, 1, 0.5)
	dc.DrawLine(20, 0, 18, 0)
	dc.DrawStringAnchored("E", 24, 0, 0, 0.5)
	dc.DrawLine(-20, 0, -18, 0)
	dc.DrawStringAnchored("S", 0, 24, 0.5, 1)
	dc.DrawLine(0, 20, 0, 18)
	windAngle := s1.Angle(weatherData.WindDirection) * s1.Degree
	dc.Rotate(windAngle.Radians())
	dc.DrawLine(0, -17.5, 0, 17.5)
	dc.DrawLine(0, 17.5, 5, 12.5)
	dc.DrawLine(0, 17.5, -5, 12.5)
	dc.Stroke()
	dc.Pop()
	dc.DrawStringAnchored(string(location), 10, 290, 0, 0)
	dc.DrawStringAnchored(weatherData.Time.Format(time.RFC3339), 290, 290, 1, 0)
	return dc.SavePNG(cfg.OutputFile)
}
