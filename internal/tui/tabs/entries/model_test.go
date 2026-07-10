package entries
import ("testing"; tea "charm.land/bubbletea/v2"; "charm.land/lipgloss/v2"; zone "github.com/lrstanley/bubblezone/v2"; "github.com/madicen/naitv-mcp/internal/store"; "github.com/madicen/naitv-mcp/pkg/entry")
func pressKey(s string) tea.KeyPressMsg { if len(s)==1 { r:=rune(s[0]); return tea.KeyPressMsg(tea.Key{Text:s,Code:r,ShiftedCode:r}) }; return tea.KeyPressMsg{Text:s} }
func TestBuildFlatItems_GroupCollapse(t *testing.T) {
	m := NewModel(zone.New()); m.SetDimensions(120,40)
	m.entries = []entry.Entry{{Kind:"rule",Name:"a",ProposedBy:"plugin:alpha"},{Kind:"rule",Name:"b",ProposedBy:"plugin:alpha"},{Kind:"note",Name:"c"}}
	m.buildFlatItems(); hi:=-1
	for i,it := range m.flatItems { if it.kind==itemKindHeader && it.groupName=="alpha" { hi=i } }
	if hi<0 { t.Fatal("no header") }
	m.collapsed["alpha"]=true; m.buildFlatItems()
	for _,it := range m.flatItems { if it.kind==itemKindEntry && it.groupName=="alpha" { t.Fatal("hidden") } }
}
func TestUpdate_SearchEnterEmitsRequest(t *testing.T) {
	m := NewModel(zone.New()); m.searchMode=true; m.searchInput.SetValue("findme")
	u,req,_ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text:"enter"})
	if req==nil || !req.Search { t.Fatalf("%#v",req) }
	if u.searchMode { t.Fatal("search open") }
}
func TestUpdate_ToggleDeliveryRequest(t *testing.T) {
	m := NewModel(zone.New()); m.entries=[]entry.Entry{{Kind:"note",Name:"d1",ID:"id1"}}; m.buildFlatItems()
	_,req,_ := m.Update(pressKey("i")); if req==nil || !req.ToggleDelivery { t.Fatalf("%#v",req) }
}
func TestUpdate_CopyBodyRequest(t *testing.T) {
	m := NewModel(zone.New()); m.entries=[]entry.Entry{{Kind:"note",Name:"c1",ID:"id1",Body:"p"}}; m.buildFlatItems()
	_,req,_ := m.Update(pressKey("c")); if req==nil || !req.CopyBody { t.Fatalf("%#v",req) }
}
func TestToggleDeliveryCmd(t *testing.T) {
	st,_ := store.Open(t.TempDir()+"/t.db"); defer st.Close()
	e,_ := st.Create(entry.Entry{Kind:"note",Name:"d",Body:"x"})
	ToggleDeliveryCmd(st,e.ID,"")()
	g,_ := st.Get(e.ID)
	if g.DeliveryOrDefault()!=entry.DeliveryOnDemand { t.Fatal() }
}
func TestSearchCmd(t *testing.T) {
	st,_ := store.Open(t.TempDir()+"/t.db"); defer st.Close()
	if _, err := st.Create(entry.Entry{Kind:"repo",Name:"find-me",Body:"needle"}); err != nil { t.Fatal(err) }
	msg := SearchCmd(st,"needle")().(SearchResultsMsg)
	if len(msg.Entries)!=1 || msg.Entries[0].Name!="find-me" { t.Fatal() }
}
func TestViewFitsHeight(t *testing.T) {
	for _,h := range []int{10,24,50} { m:=NewModel(zone.New()); m.SetDimensions(80,h); if lipgloss.Height(m.View())>h { t.Fatal() } }
}
