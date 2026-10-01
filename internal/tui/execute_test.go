package tui

import (
 "context"
 "fmt"
 "net/http"
 "os"
 "path/filepath"
 "strings"
 "testing"

 "apitool/internal/app"
 "apitool/internal/auth"
 "apitool/internal/model"
 tea "github.com/charmbracelet/bubbletea"
)

func requestScreen(t *testing.T, method string, confirm bool) (Model, *int) {
 t.Helper()
 root := t.TempDir()
 for path, data := range map[string]string{
 ".git/keep":"", "demo/.api/collection.yaml":"name: Demo\n", "demo/.api/environments/test.yaml":"name: test\nvariables:\n  host: https://example.test\n", "demo/.api/requests/item.yaml":fmt.Sprintf("name: Item\nmethod: %s\nrequest:\n  url: '{{host}}/items?access_token=secret'\n",method),
 } { full:=filepath.Join(root,path); if err:=os.MkdirAll(filepath.Dir(full),0755);err!=nil {t.Fatal(err)};if err:=os.WriteFile(full,[]byte(data),0644);err!=nil {t.Fatal(err)} }
 calls:=new(int)
 service,_:=app.New(app.Dependencies{Execute:func(_ context.Context,e model.EffectiveRequest,_ auth.TokenProvider)(model.Response,*model.ExecutionError){*calls++;if e.Method!=method {t.Errorf("method=%s",e.Method)};return model.Response{StatusCode:401,Headers:http.Header{"Content-Type":{"application/json"}},Body:[]byte(`{"error":"denied"}`)},nil}})
 if _,err:=service.OpenWorkspace(context.Background(),root,app.OpenOptions{Collection:"demo",Environment:"test",ConfirmDangerous:confirm});err!=nil {t.Fatal(err)}
 m:=New(service,Options{StartingCollection:"demo",ConfirmDangerous:confirm}).(Model)
 m.focus=requestPane
 return m,calls
}
func sendKey() tea.KeyMsg { return tea.KeyMsg{Type:tea.KeyCtrlJ} }
func TestConfirmDangerousRequiresExplicitSendForDelete(t *testing.T){
 m,calls:=requestScreen(t,http.MethodDelete,true);next,cmd:=m.Update(sendKey())
 if !strings.Contains(next.View(),"Confirmation required")||strings.Contains(next.View(),"Sending")||cmd!=nil||*calls!=0 {t.Fatalf("view=%s cmd=%v calls=%d",next.View(),cmd,*calls)}
}
func TestConfirmationAcceptSendsOnceCancelSendsNone(t *testing.T){
 for _,method:=range []string{"POST","PUT","PATCH","DELETE"}{t.Run(method,func(t *testing.T){
 m,calls:=requestScreen(t,method,true);next,_:=m.Update(sendKey());if strings.Contains(next.View(),"secret")||!strings.Contains(next.View(),"example.test/items") {t.Fatalf("unsafe/unresolved modal: %s",next.View())}
 canceled,cmd:=next.Update(tea.KeyMsg{Type:tea.KeyEsc});if cmd!=nil||*calls!=0||strings.Contains(canceled.View(),"Confirmation required"){t.Fatal("cancel sent")}
 next,_=canceled.Update(sendKey());next,_=next.Update(tea.KeyMsg{Type:tea.KeyTab});next,cmd=next.Update(tea.KeyMsg{Type:tea.KeyEnter});if cmd==nil||!strings.Contains(next.View(),"Sending"){t.Fatal("accept did not start send")}
 duplicate,second:=next.Update(sendKey());if second!=nil {t.Fatal("duplicate send command")};next,_=duplicate.Update(cmd());if *calls!=1||!strings.Contains(next.View(),"401 Unauthorized"){t.Fatalf("calls=%d view=%s",*calls,next.View())}
 })}
}
func TestGETSendsImmediatelyAndEnterActivatesSend(t *testing.T){for _,key:=range []tea.KeyMsg{sendKey(),{Type:tea.KeyEnter}}{m,calls:=requestScreen(t,"GET",true);next,cmd:=m.Update(key);if cmd==nil||strings.Contains(next.View(),"Confirmation required"){t.Fatal("GET blocked")};next,_=next.Update(cmd());if *calls!=1 {t.Fatal("GET count",*calls)}}}
