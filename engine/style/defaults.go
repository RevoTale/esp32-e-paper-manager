package style

import "image/color"

func defaultValues() map[string]Value {
	values := backgroundDefaults()
	keywords := map[string]string{
		"display": "block", "position": "static", "box-sizing": "content-box", "overflow": "visible",
		"font-family": "go", "font-style": "normal", "font-weight": "normal", "line-height": "normal",
		"text-align": "left", "white-space": "normal", "overflow-wrap": "normal", "text-overflow": "clip",
		"object-fit": "fill", "z-index": "auto", "aspect-ratio": "auto",
	}
	for name, value := range keywords {
		values[name] = Value{Kind: KeywordValue, Keyword: value}
	}
	for _, name := range [...]string{"width", "height", "top", "right", "bottom", "left"} {
		values[name] = Value{Kind: LengthValue, Length: Length{Unit: Auto}}
	}
	for _, name := range [...]string{"max-width", "max-height"} {
		values[name] = Value{Kind: LengthValue, Length: Length{Unit: None}}
	}
	for _, name := range [...]string{"min-width", "min-height", "border-radius"} {
		values[name] = Value{Kind: LengthValue}
	}
	for _, name := range [...]string{"object-position-x", "object-position-y"} {
		values[name] = Value{Kind: LengthValue, Length: Length{Value: 50, Unit: Percent}}
	}
	for _, side := range sides {
		values["padding-"+side] = Value{Kind: LengthValue}
		values["margin-"+side] = Value{Kind: LengthValue}
		values["border-"+side+"-width"] = Value{Kind: LengthValue, Length: Length{Value: 3}}
		values["border-"+side+"-style"] = Value{Kind: KeywordValue, Keyword: "none"}
		values["border-"+side+"-color"] = Value{Kind: ColorValue, Keyword: "currentcolor"}
	}
	values["color"] = Value{Kind: ColorValue, Color: color.NRGBA{A: 255}}
	values["font-size"] = Value{Kind: LengthValue, Length: Length{Value: 16}}
	values["opacity"] = Value{Kind: NumberValue, Number: 1}
	return values
}

func semanticDefaults(values map[string]Value, tag string) {
	switch tag {
	case "span", "strong", "b", "em", "i", "small", "code", "br", "img":
		values["display"] = Value{Kind: KeywordValue, Keyword: "inline"}
	case "head", "title", "meta":
		values["display"] = Value{Kind: KeywordValue, Keyword: "none"}
	}
	switch tag {
	case "strong", "b", "h1", "h2", "h3", "h4", "h5", "h6":
		values["font-weight"] = Value{Kind: KeywordValue, Keyword: "bold"}
	case "em", "i":
		values["font-style"] = Value{Kind: KeywordValue, Keyword: "italic"}
	}
	semanticMetrics(values, tag)
}

func semanticMetrics(values map[string]Value, tag string) {
	sizes := map[string]float64{"h1": 32, "h2": 24, "h3": 20, "h4": 18, "h5": 16, "h6": 16}
	if size, ok := sizes[tag]; ok {
		values["font-size"] = Value{Kind: LengthValue, Length: Length{Value: size}}
	}
	switch tag {
	case "small":
		values["font-size"] = Value{Kind: LengthValue, Length: Length{Value: .8, Unit: Em}}
	case "pre", "code":
		values["font-family"] = Value{Kind: KeywordValue, Keyword: "monospace"}
	case "ul", "ol":
		values["padding-left"] = Value{Kind: LengthValue, Length: Length{Value: 24}}
	case "hr":
		values["border-top-style"] = Value{Kind: KeywordValue, Keyword: "solid"}
		values["border-top-width"] = Value{Kind: LengthValue, Length: Length{Value: 1}}
	}
	if tag == "pre" {
		values["white-space"] = Value{Kind: KeywordValue, Keyword: "pre"}
	}
}
