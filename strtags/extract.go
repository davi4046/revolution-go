package strtags

import "strings"

func Extract(s string) []Tag {

	var tags []Tag

	tagStrings := strings.Split(s, "@")[1:]
	for _, tagString := range tagStrings {
		name, optionsString, _ := strings.Cut(tagString, " ")

		options := strings.Split(optionsString, ",")

		tags = append(tags, Tag{
			Name:    name,
			Options: options,
		})
	}
	return tags
}
