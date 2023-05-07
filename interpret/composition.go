package interpreter

type Composition struct {
	Key struct {
		Root string `xml:"root,attr"`
		Mode string `xml:"mode,attr"`
	}
	KeyChanges struct {
		KeyChange []struct {
			Beat float64 `xml:"beat,attr"`
			Key  struct {
				Root string `xml:"root,attr"`
				Mode string `xml:"mode,attr"`
			}
		}
	}
	Definitions struct {
		GenDef []struct {
			GenId string `xml:"genId,attr"`
		}
		ModDef []struct {
			ModId string `xml:"modId,attr"`
		}
	}
	Channels struct {
		Channel []struct {
			Instrument string `xml:"instrument,attr"`
			Track      []struct {
				Item []struct {
					Beat    float64 `xml:"beat,attr"`
					Length  float64 `xml:"length,attr"`
					Content string
				}
			}
		}
	}
}
