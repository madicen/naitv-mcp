package plugins
import ("testing"; "charm.land/lipgloss/v2"; zone "github.com/lrstanley/bubblezone/v2")
func TestViewFitsHeight(t *testing.T) { for _,h := range []int{10,24,50} { m:=NewModel(zone.New()); m.SetDimensions(80,h); if lipgloss.Height(m.View())>h { t.Fatal() } } }
