package gobay

import (
    "bytes"
    "errors"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
)


type EbayCall struct {
    DevID              string
    AppID              string
    CertID             string
    CompatLevel        string
    SiteID             string
    EndPoint           string
    EbayAuthToken      string
    Country            string
    Currency           string
    Language           string
    MessageID          string
    WarningLevel       string
    PayPalEmailAddress string
    Callname           string
    XMLData            string
    Headers            map[string]string
    Products           []Item
    TheClient          *http.Client
    CategoryCallInfo   GetCategoriesStruct
}

func NewEbayCallEx(conf []byte) (*EbayCall, error) {
    var e EbayCall
    m := make(map[string]string)
    c := make(map[interface{}]interface{})
    err := LoadConfiguration(conf, &c)

    if err != nil {
        return nil, err
    }

    e.DevID = c["DevID"].(string)
    e.AppID = c["AppID"].(string)
    e.CertID = c["CertID"].(string)
    e.CompatLevel = c["CompatLevel"].(string)
    e.SiteID = c["SiteID"].(string)
    e.EndPoint = c["EndPoint"].(string)
    e.EbayAuthToken = c["EbayAuthToken"].(string)
    e.Country = c["Country"].(string)
    e.Currency = c["Currency"].(string)
    e.PayPalEmailAddress = c["PayPalEmailAddress"].(string)
    e.Language = c["Language"].(string)
    e.WarningLevel = c["WarningLevel"].(string)

    m["X-EBAY-API-COMPATIBILITY-LEVEL"] = fmt.Sprintf("%s", e.CompatLevel)
    m["X-EBAY-API-DEV-NAME"] = fmt.Sprintf("%s", e.DevID)
    m["X-EBAY-API-APP-NAME"] = fmt.Sprintf("%s", e.AppID)
    m["X-EBAY-API-CERT-NAME"] = fmt.Sprintf("%s", e.CertID)
    //m["X-EBAY-API-CALL-NAME"] = fmt.Sprintf("%s", e.CallName)
    m["X-EBAY-API-SITEID"] = fmt.Sprintf("%s", e.SiteID)
    e.Headers = m

    return &e, nil
}

func (o *EbayCall) NewItem() *Item {
    p := NewItem()
    p.Country = o.Country
    p.Site = SiteIDToCode(o.SiteID)
    p.Currency = o.Currency
    p.PayPalEmailAddress = o.PayPalEmailAddress
    return p
}

func (o *EbayCall) SetHeader(k string, v string) {
    o.Headers[k] = v
}
func (o *EbayCall) GetHeader(k string) string {
    return o.Headers[k]
}
func (o *EbayCall) SetCallname(v string) {
    o.Headers["X-EBAY-API-CALL-NAME"] = v
}
func (o *EbayCall) GetCallname() string {
    return o.Headers["X-EBAY-API-CALL-NAME"]
}
func (o *EbayCall) Execute(r *[]Result) error {
    cl := o.GetCallname()
    if cl == "GeteBayOfficialTime" {
        err := o.GeteBayOfficialTime(r)
        if err != nil {
            return err
        }
        return o.Send(r)
    }

    if cl == "GetAllCategories" {
        o.SetCallname("GetCategories")
        err := o.GetAllCategories(r)
        if err != nil {
            return err
        }
        return o.Send(r)
    }

    return nil
}
func (o *EbayCall) Send(r *[]Result) error {
    o.TheClient = new(http.Client)

    globalDebugFunction(DBG_DEBUG, fmt.Sprintf("About to send [[%s]]\n\n", o.XMLData))

    if o.XMLData == "" {
        err := errors.New("XMLData was empty!")
        e := NewFakeResult(fmt.Sprintf("%s", err))
        *r = append(*r, *e)
        return err
    }
    req, err := http.NewRequest("POST", o.EndPoint, bytes.NewBufferString(o.XMLData))
    if err != nil {
        e := NewFakeResult(fmt.Sprintf("%s", err))
        *r = append(*r, *e)
        return err
    }
    //req.URL.Host = "148.251.124.116:9090"
    for k, v := range o.Headers {
        req.Header.Set(k, v)
    }

    resp, err := o.TheClient.Do(req)
    //Post(o.EndPoint, "text/xml; charset=utf-8", )
    if err != nil {
        e := NewFakeResult(fmt.Sprintf("%s", err))
        *r = append(*r, *e)
        return err
    }
    //fmt.Printf("%+v\n", resp)
    b, err := ioutil.ReadAll(resp.Body)
    defer resp.Body.Close()
    // We should cache the results of certain calls

    globalDebugFunction(DBG_DEBUG, fmt.Sprintf("[[BODY: %s]]", string(b)))
    if err != nil {
        e := NewFakeResult(fmt.Sprintf("%s", err))
        *r = append(*r, *e)
        return err
    }
    if !fileExists(".cache") {
        err = os.Mkdir(".cache", 0777)
        if err != nil {
            globalDebugFunction(DBG_WARN, "Could not create .cache")
        } else {
            filePutContents(fmt.Sprintf(".cache/%s-%s.xml", o.GetCallname(), o.MessageID), string(b))
        }
    }
    res, err := NewResult(b)
    if err != nil {
        res = NewFakeResult(fmt.Sprintf("%s", err))
    }
    *r = append(*r, *res)
    return nil
}
func (o *EbayCall) GeteBayOfficialTime(r *[]Result) error {
    o.MessageID, _ = pseudoUUID()
    body, err := compileGoString("Time", GeteBayOfficialTimeTemplate(), o, nil)
    if err != nil {
        return err
    }
    final_xml, err := compileGoString("FinalTime", WrapCall("GeteBayOfficialTime", "", body, ""), o, nil)
    if err != nil {
        return err
    }
    o.XMLData = final_xml
    return nil
}

func (o *EbayCall) GetAllCategories(r *[]Result) error {
    o.MessageID, _ = pseudoUUID()
    o.CategoryCallInfo.SiteID = o.SiteID
    if o.CategoryCallInfo.LevelLimit == "" {
        o.CategoryCallInfo.LevelLimit = "3"
    }
    if o.CategoryCallInfo.ViewAllNodes != "true" {
        o.CategoryCallInfo.ViewAllNodes = "false"
    }

    body, err := compileGoString("Time", GetAllCategoriesTemplate(), o.CategoryCallInfo, nil)
    if err != nil {
        return err
    }
    final_xml, err := compileGoString("FinalGetAllCategories", WrapCall("GetCategories", "", body, ""), o, nil)
    if err != nil {
        return err
    }
    o.XMLData = final_xml
    return nil
}


// Debug Functions