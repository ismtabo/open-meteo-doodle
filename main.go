package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"log"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"golang.org/x/image/font"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const APP_NAME = "open-meteo"
const DEFAULT_TIMEZONE = "Europe/Madrid"

type Config struct {
	Timezone   string
	Verbose    bool
	OutputFile string `mapstructure:"file"`
}

type Location = time.Location

func init() {
	flag.String("file", "", "Output file")
	flag.BoolP("verbose", "v", false, "Verbose mode")
}

func main() {
	flag.Parse()
	viper.BindPFlags(flag.CommandLine)
	cfg := Config{}
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatal("Error: failed unmarshaling config", err)
	}
	if !cfg.Verbose {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		slog.SetDefault(logger)
	}
	if cfg.OutputFile == "" {
		log.Fatal("Error: missing output file option")
	}
	weatherData, err := retrieveOpenWeatherCurrentWeather()
	if err != nil {
		log.Fatal("Error: failed retrieving data", err)
	}
	if err := drawImage(cfg, weatherData); err != nil {
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

const OPEN_WEATHER_URL = "https://api.open-meteo.com/v1/forecast?latitude=41.6552&longitude=-4.7237&current=temperature_2m,relative_humidity_2m,apparent_temperature,is_day,precipitation_probability,weather_code,wind_speed_10m,wind_direction_10m&timezone=Europe%2FMadrid&forecast_days=1&forecast_hours=24"

func retrieveOpenWeatherCurrentWeather() (WeatherData, error) {
	req, err := http.Get(OPEN_WEATHER_URL)
	if err != nil {
		return WeatherData{}, fmt.Errorf("error requesting info: %w", err)
	}
	buff := &bytes.Buffer{}
	io.Copy(buff, req.Body)
	slog.Info("response body", slog.String("body", buff.String()))
	data := OpenWeatherData{}
	if err := json.NewDecoder(buff).Decode(&data); err != nil {
		return WeatherData{}, fmt.Errorf("error decoding response: %w", err)
	}
	loc, err := time.LoadLocation(data.Timezone)
	if err != nil {
		return WeatherData{}, fmt.Errorf("error loading timezone: %w", err)
	}
	t, err := time.ParseInLocation("2006-01-02T15:04", data.Current.Time, loc)
	if err != nil {
		return WeatherData{}, fmt.Errorf("error decoding time: %w", err)
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
const UBUNTU_FONT_PATH = "/usr/share/fonts/truetype/ubuntu/Ubuntu-Th.ttf"
const TERMUX_FONT_PATH = "/data/data/com.termux/files/usr/share/fonts/TTF/DejaVuSans.ttf"

var fonts = map[string]string{
	"linux":   UBUNTU_FONT_PATH,
	"android": TERMUX_FONT_PATH,
}

const UBUNTU_EMOJI_FONT_PATH = "/home/cx02962/.cache/.fr-KwXqja/NotoEmoji-VariableFont_wght.ttf"

//go:embed fonts/NotoEmoji-VariableFont_wght.ttf
var emojiFontTTF []byte

func loadEmojiFont(points float64) (font.Face, error) {
	f, err := truetype.Parse(emojiFontTTF)
	if err != nil {
		return nil, fmt.Errorf("error parsing emoji font: %w", err)
	}
	face := truetype.NewFace(f, &truetype.Options{
		Size: points,
		// Hinting: font.HintingFull,
	})
	return face, nil
}

var wmoCodeToEmoji = map[int]string{
	0:  "☀",  // Clear sky
	1:  "🌤️", // Mainly clear
	2:  "⛅",  // Partly cloudy
	3:  "☁️", // Overcast
	45: "🌫️", // Fog
	48: "🌫️", // Depositing rime fog
	51: "🌧️", // Drizzle: Light intensity
	53: "🌧️", // Drizzle: Moderate intensity
	55: "🌧️", // Drizzle: Dense intensity
	56: "🌧️", // Freezing Drizzle: Light intensity
	57: "🌧️", // Freezing Drizzle: Dense intensity
	61: "🌧️", // Rain: Slight intensity
	63: "🌧️", // Rain: Moderate intensity
	65: "🌧️", // Rain: Heavy intensity
	66: "🌧️", // Freezing Rain: Light intensity
	67: "🌧️", // Freezing Rain: Heavy intensity
	71: "❄️", // Snow fall: Slight intensity
	73: "❄️", // Snow fall: Moderate intensity
	75: "❄️", // Snow fall: Heavy intensity
	77: "❄️", // Snow grains
	80: "🌧️", // Rain showers: Slight intensity
	81: "🌧️", // Rain showers: Moderate intensity
	82: "🌧️", // Rain showers: Violent intensity
	85: "❄️", // Snow showers: Slight intensity
	86: "❄️", // Snow showers: Heavy intensity
	95: "⛈️", // Thunderstorm: Slight or moderate
	96: "⛈️", // Thunderstorm with slight hail
	99: "⛈️", // Thunderstorm with heavy hail
}

func drawImage(cfg Config, weatherData WeatherData) error {
	uname := runtime.GOOS
	fontPath, ok := fonts[uname]
	if !ok {
		return fmt.Errorf("unknown platform %s", uname)
	}
	emojiFont, err := loadEmojiFont(150)
	if err != nil {
		return fmt.Errorf("error loading emoji font: %w", err)
	}
	w := SIZE
	h := SIZE
	dc := gg.NewContext(w, h)
	fgColor := color.White
	if weatherData.IsDay {
		fgColor = color.Gray16{0x1111}
	}
	bgColor := color.Gray16{0x1111}
	if weatherData.IsDay {
		bgColor = color.White
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
	dc.SetFontFace(emojiFont)
	dc.DrawStringAnchored(wmoCodeToEmoji[weatherData.WeatherCode], 150, 150, 0.5, 0.75)
	smallFS := fs * 0.3
	dc.LoadFontFace(fontPath, smallFS)
	dc.DrawStringAnchored(weatherData.Time.Format(time.RFC3339), 290, 290, 1, 0)
	return dc.SavePNG(cfg.OutputFile)
}
