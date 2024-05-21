package m3u

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// MakeInterfaceFromM3U :
func MakeInterfaceFromM3U(byteStream []byte) (allChannels []interface{}, err error) {

	var content = string(byteStream)
	var channelName string
	var uuids []string

	var parseMetaData = func(channel string) (stream map[string]string) {

		stream = make(map[string]string)
		var exceptForParameter = `[a-zA-Z&=]*(".*?")`
		var exceptForChannelName = `,([^\n]*|,[^\r]*)`
		var lines = strings.Split(strings.Replace(channel, "\r\n", "\n", -1), "\n")
		var comments []string // Initialize comments slice for each channel
		// Remove empty lines
		for i := len(lines) - 1; i >= 0; i-- {
			if len(lines[i]) == 0 {
				lines = append(lines[:i], lines[i+1:]...)
			}
		}

		if len(lines) >= 2 {
			for _, line := range lines {
				if len(line) > 0 && line[0:1] == "#" {
					comments = append(comments, strings.TrimSpace(line)) // Add comment to comments slice
					continue
				}

				_, err := url.ParseRequestURI(line)
				switch err {
				case nil:
					stream["url"] = strings.Trim(line, "\r\n")

				default:
					var value string
					// Parse all parameters
					var p = regexp.MustCompile(exceptForParameter)
					var streamParameter = p.FindAllString(line, -1)
					for _, p := range streamParameter {
						line = strings.Replace(line, p, "", 1)
						p = strings.Replace(p, `"`, "", -1)
						var parameter = strings.SplitN(p, "=", 2)
						if len(parameter) == 2 {
							// Store TVG key in lowercase
							if strings.Contains(parameter[0], "tvg") {
								stream[strings.ToLower(parameter[0])] = parameter[1]
							} else {
								stream[parameter[0]] = parameter[1]
							}

							// Don't pass URLs to the filter function
							if !strings.Contains(parameter[1], "://") && len(parameter[1]) > 0 {
								value = value + parameter[1] + " "
							}
						}
					}

					// Parse channel name
					n := regexp.MustCompile(exceptForChannelName)
					var name = n.FindAllString(line, 1)
					if len(name) > 0 {
						channelName = name[0]
						channelName = strings.Replace(channelName, `,`, "", 1)
						channelName = strings.TrimRight(channelName, "\r\n")
						channelName = strings.TrimRight(channelName, " ")
					}

					if len(channelName) == 0 {
						if v, ok := stream["tvg-name"]; ok {
							channelName = v
						}
					}

					channelName = strings.TrimRight(channelName, " ")

					// Skip channels without names
					if len(channelName) == 0 {
						return
					}

					stream["name"] = channelName
					value = value + channelName

					stream["_values"] = value
				}
			}
		}

		// Look for unique ID in the stream
		for key, value := range stream {
			if strings.Contains(strings.ToLower(key), "tvg-id") {
				if indexOfString(value, uuids) != -1 {
					break
				}
				uuids = append(uuids, value)
				stream["_uuid.key"] = key
				stream["_uuid.value"] = value
				break
			}
		}

		// Add comments to the stream
		if len(comments) > 0 {
			stream["comments"] = strings.Join(comments, "\n")
		}

		return
	}

	if strings.Contains(content, "#EXT-X-TARGETDURATION") || strings.Contains(content, "#EXT-X-MEDIA-SEQUENCE") {
		err = errors.New("Invalid M3U file, an extended M3U file is required.")
		return
	}

	if strings.Contains(content, "#EXTM3U") {
		content = strings.Replace(content, ":-1", "", -1)
		content = strings.Replace(content, "'", "\"", -1)
		var channels = strings.Split(content, "#EXTINF")
		channels = append(channels[:0], channels[1:]...)

		for _, channel := range channels {
			var stream = parseMetaData(channel)
			if len(stream) > 0 && stream != nil {
				allChannels = append(allChannels, stream)
			}
		}
	} else {
		err = errors.New("Invalid M3U file, an extended M3U file is required.")
	}

	return
}

func indexOfString(element string, data []string) int {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1
}
