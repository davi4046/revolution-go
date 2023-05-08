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
			Id string `xml:"id,attr"`
		}
		ModDef []struct {
			Id string `xml:"id,attr"`
		}
	}
	Channels struct {
		Channel []struct {
			Instrument string `xml:"instrument,attr"`
			Track      []struct {
				Item []struct {
					Beat   float64 `xml:"beat,attr"`
					Length float64 `xml:"length,attr"`
					Gen    struct {
						Ref   string  `xml:"ref,attr"`
						Start float64 `xml:"start,attr"`
						Add   int     `xml:"add,attr"`
						Sub   int     `xml:"sub,attr"`
					}
					Mod struct {
						Ref string `xml:"ref,attr"`
					}
				}
			}
		}
	}
}
