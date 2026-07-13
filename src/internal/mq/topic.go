package mq

import "strings"

// BuildTopic builds a topic using /{direction}/{deviceID}/{prefix}.
func BuildTopic(direction, deviceID, prefix string) string {
	d := strings.Trim(direction, "/")
	dev := strings.Trim(deviceID, "/")
	p := strings.Trim(prefix, "/")

	if p == "" {
		return "/" + d + "/" + dev
	}
	return "/" + d + "/" + dev + "/" + p
}
