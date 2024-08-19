# Open Meteo Doodle

Open Meteo Doodle too generates a doodle based on the current meteorology information for a given location. It retrieves
the information from free [Open Meteo API][open-meteo-api], and the location city name from
[Open Street Map Nominatim][nominatim].

[open-meteo-api]: https://open-meteo.com/
[nominatim]: https://nominatim.openstreetmap.org/ui/about.html

![sample.png](/docs/imgs/sample.png)

Information of the doodle:

- Temperature
- Humidity
- Probability of Precipitation
- Wind speed and direction
- Location city name

## Usage

First, install the CLI tool:

```console
go install github.com/ismtabo/open-meteo-doodle@latest
```

Then, use the tool to generate the doodle in the given file:

```console
open-meteo-doodle [options]
```

### Options

- `-c, --config <config>`: Configuration file
- `--latitude <latitude>`: Latitude for the desired location
- `--longitude <longitude>`: Longitude for the desired location
- `--file <file>`: Output file
- `-v,--verbose`: Verbose mode

### Configuration

The configuration file has the following properties:

- `latitude`: Latitude for the desired location
- `longitude`: Longitude for the desired location
- `file`: Output file

For example:

```yaml
# Valladolid (Spain)
latitude: 41.6552
longitude: -4.7237
file: output.png
verbose: true
```

## Authors

- Ismael Taboada Rodero: @ismtabo

## Acknowledgment

- Noto Emoji: https://fonts.google.com/noto/specimen/Noto+Emoji
- Open Meteo API: https://open-meteo.com/
- OpenStreetMap Nominatim: https://nominatim.openstreetmap.org/ui/about.html
