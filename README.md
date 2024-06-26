# Open Meteo Doodle

Open Meteo Doodle too generates a doodle based on the current meteo information
of Valladolid (41°39′07″N 4°43′43″O). It retrieves the information from free
[Open Meteo API][open-meteo-api].

[open-meteo-api]: https://open-meteo.com/

![sample.png](/docs/imgs/sample.png)

Information of the doodle:

- Temperature
- Humidity
- Probability of Precipitation
- Wind speed and direction

## Usage

First, install the CLI tool:

```console
go install github.com/ismtabo/open-meteo-doodle
```

Then, use the tool to generate the doodle in the given file:

```console
open-meteo-doodle --file <file>
```

## Authors

- Ismael Taboada Rodero: @ismtabo

## Acknowledgment

- Noto Emoji: https://fonts.google.com/noto/specimen/Noto+Emoji
- Open Meteo API: https://open-meteo.com/
- OpenStreetMap Nominatim: https://nominatim.openstreetmap.org/ui/about.html
