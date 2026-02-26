package ui

const asciiArt = `    ____  ___    __    ____  __  __
   / __ \/   |  / /   / __ \/ / / /
  / /_/ / /| | / /   / /_/ / /_/ /
 / _, _/ ___ |/ /___/ ____/ __  /
/_/ |_/_/  |_/_____/_/   /_/ /_/`

// Banner returns the ASCII art banner rendered in Simpsons Yellow.
func (t *Theme) Banner() string {
	return t.BannerStyle.Render(asciiArt)
}
