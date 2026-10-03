package style

import (
	"image/color"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

// Computed owns resolved inheritance and font metrics for one element. Layout
// still resolves percentages and box-relative lengths against its containing box.
type Computed struct {
	values   map[string]Value
	rootFont float64
}

func (c Computed) Get(name string) Value         { return c.values[name] }
func (c Computed) Length(name string) Length     { return c.values[name].Length }
func (c Computed) Number(name string) float64    { return c.values[name].Number }
func (c Computed) Keyword(name string) string    { return c.values[name].Keyword }
func (c Computed) Color(name string) color.NRGBA { return c.values[name].Color }
func (c Computed) FontSize() float64             { return c.Length("font-size").Value }
func (c Computed) RootFont() float64             { return c.rootFont }

func Compute(tag string, block Block, parent *Computed, metrics Metrics) (Computed, error) {
	c := Computed{values: defaultValues(), rootFont: 16}
	metrics.Font, metrics.RootFont = 16, 16
	if parent != nil {
		c.inherit(*parent)
		metrics.Font, metrics.RootFont = parent.FontSize(), parent.rootFont
	}
	if tag == "html" {
		metrics.RootFont = 16
	}
	semanticDefaults(c.values, tag)
	for name, entry := range block.values {
		c.values[name] = entry.value
	}
	c.resolveDisplay()
	metrics.Containing = metrics.Font // font-size % is relative to inherited font.
	size, err := c.Length("font-size").Resolve(metrics)
	if err != nil || size < 8 || size > 128 {
		return Computed{}, block.Error("font-size", renderdiag.InvalidValue)
	}
	c.values["font-size"] = Value{Kind: LengthValue, Length: Length{Value: size}}
	if parent == nil || tag == "html" {
		c.rootFont = size
	}
	metrics.Font, metrics.RootFont, metrics.Containing = size, c.rootFont, size
	if err := c.resolveLineHeight(metrics); err != nil {
		return Computed{}, block.Error("line-height", renderdiag.InvalidValue)
	}
	c.resolveColors(parent)
	return c, nil
}

func (c *Computed) resolveDisplay() {
	// Absolute positioning blockifies inline/inline-block, but not display:none.
	// https://www.w3.org/TR/CSS22/visuren.html#dis-pos-flo
	if c.Keyword("position") == "absolute" && c.Keyword("display") != "none" {
		c.values["display"] = Value{Kind: KeywordValue, Keyword: "block"}
	}
}

func (c *Computed) inherit(parent Computed) {
	for _, name := range [...]string{"color", "font-family", "font-size", "font-style", "font-weight", "line-height", "text-align", "white-space", "overflow-wrap"} {
		c.values[name] = parent.values[name]
	}
	c.rootFont = parent.rootFont
}

func (c *Computed) resolveLineHeight(metrics Metrics) error {
	value := c.values["line-height"]
	if value.Kind != LengthValue {
		return nil
	}
	height, err := value.Length.Resolve(metrics)
	if err != nil || height <= 0 {
		return ErrValue
	}
	c.values["line-height"] = Value{Kind: LengthValue, Length: Length{Value: height}}
	return nil
}

func (c *Computed) resolveColors(parent *Computed) {
	foreground := c.values["color"]
	if foreground.Keyword == "currentcolor" {
		foreground = Value{Kind: ColorValue, Color: color.NRGBA{A: 255}}
		if parent != nil {
			foreground.Color = parent.Color("color")
		}
		c.values["color"] = foreground
	}
	for name, value := range c.values {
		if value.Kind == ColorValue && value.Keyword == "currentcolor" {
			c.values[name] = foreground
		}
	}
}
