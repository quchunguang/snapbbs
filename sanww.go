// Scratch BBS to local HTML folder
// Tested on phpwind 1.8.7
package main

import (
	// "encoding/json"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/axgle/mahonia"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// 要抓取的板块ID (fid)
var fids = []int{166}

const (
	BASE_URL   = "http://bbs.sanww.com"
	URL_T      = BASE_URL + "/read.php?tid-%d.html"
	URL_T_PAGE = BASE_URL + "/read.php?tid-%d-page-%d.html"
	URL_F      = BASE_URL + "/thread.php?fid-%d.html"
	URL_F_PAGE = BASE_URL + "/thread.php?fid-%d-page-%d.html"
)

// Global decoder from gb18030
var ds func(s string) string

type Follow struct {
	Author   string
	AuthorId int
	PostTime string
	Html     string
	Text     string
}

type T struct {
	Id      int
	Title   string
	Page    int
	Follows []Follow
}

func GetT(id int) (t *T, err error) {
	url := fmt.Sprintf(URL_T, id)
	log.Printf("%-10s%s\n", "[Access]", url)
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return
	}

	t = new(T)
	t.Id = id
	t.Title = ds(doc.Find("#breadCrumb a:nth-child(7)").Text())

	// Get page number
	t.Page = GetPageNum(doc, "#main div:nth-child(3) div span.fl div.pages a")

	// Get first page
	GetTPage(doc, t)

	for page := 2; page <= t.Page; page++ {
		url = fmt.Sprintf(URL_T_PAGE, id, page)
		log.Printf("%-10s%s\n", "[Access]", url)
		doc, _ = goquery.NewDocument(url)
		GetTPage(doc, t)
	}
	return
}

func GetTPage(doc *goquery.Document, t *T) {
	reg, err := regexp.Compile("聽+")
	if err != nil {
		log.Fatal(err)
	}

	doc.Find(".read_t").Each(func(i int, s *goquery.Selection) {
		var item Follow
		item.Author = ds(s.Find("td.floot_left div.readName.b a").Text())
		item.AuthorId, _ = strconv.Atoi(s.Find("td.floot_left div.readName.b a").AttrOr("href", "")[10:])
		item.PostTime = s.Find("td.floot_right div.tipTop.s6 span").Not(".fr").AttrOr("title", "")
		htmlText, _ := s.Find("td.floot_right div.tpc_content").Html()
		item.Html = ds(htmlText)
		text := ds(s.Find("td.floot_right div.tpc_content").Text())
		item.Text = reg.ReplaceAllString(text, "\n")
		t.Follows = append(t.Follows, item)
	})
}

type Record struct {
	Id            int
	Title         string
	Author        string
	AuthorId      int
	PostTime      string
	ReplyCount    int
	ViewCount     int
	LastReply     string
	LastReplyTime string
}

type F struct {
	Id         int
	Title      string
	TitleCount string
	FullCount  string
	Page       int
	Records    []Record
}

func GetF(id int) (f *F, err error) {
	url := fmt.Sprintf(URL_F, id)
	log.Printf("%-10s%s\n", "[Access]", url)
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return
	}

	f = new(F)
	f.Id = id
	f.Title = ds(doc.Find("#sidebar div div div.hB.mb10 h2").Text())
	f.TitleCount = doc.Find("div.threadInfo.mb10 table tbody tr td:nth-child(2) p span:nth-child(3)").Text()
	f.FullCount = doc.Find("div.threadInfo.mb10 table tbody tr td:nth-child(2) p span:nth-child(5)").Text()
	f.Page = GetPageNum(doc, "div#c div.cc div div.pages a")

	// Get first page
	GetFPage(doc, f)

	// for page := 2; page <= f.Page; page++ {
	// 	url = fmt.Sprintf(URL_F_PAGE, id, page)
	// 	log.Printf("%-10s%s\n", "[Access]", url)
	// 	doc, err = goquery.NewDocument(url)
	// 	GetFPage(doc, f)
	// }
	return
}

func GetFPage(doc *goquery.Document, f *F) {
	doc.Find("#threadlist tr.tr3").Each(func(i int, s *goquery.Selection) {
		var item Record
		item.Id, _ = strconv.Atoi(s.Find("td.subject").AttrOr("id", "")[3:])
		item.Title = ds(s.Find("td.subject a.subject_t.f14").Text())
		item.Author = ds(s.Find("td:nth-child(3) a").Text())
		item.AuthorId, _ = strconv.Atoi(s.Find("td:nth-child(3) a").AttrOr("href", "")[10:])
		item.PostTime = s.Find("td:nth-child(3) p").Text()
		sReplyView := s.Find("td.num").Text()
		fmt.Sscanf(sReplyView, "%d/%d", &item.ReplyCount, &item.ViewCount)
		item.LastReply = ds(s.Find("td:nth-child(5) a").AttrOr("data-card-key", ""))
		item.LastReplyTime = s.Find("td:nth-child(5) p a").Text()
		f.Records = append(f.Records, item)
	})
}

// Get split page numbers
func GetPageNum(doc *goquery.Document, selector string) (pageNum int) {
	pageNum = 1
	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		sPage := s.Text()
		if strings.HasPrefix(sPage, "...") {
			sPage = sPage[3:]
			if nPage, errPage := strconv.Atoi(sPage); errPage == nil {
				pageNum = nPage
			}
		} else {
			if nPage, errPage := strconv.Atoi(sPage); errPage == nil {
				if nPage > pageNum {
					pageNum = nPage
				}
			}
		}
	})
	return
}

func WriteTHtml(t *T) {
	const tpl = `
<!DOCTYPE html>
<html lang="zh_CN">
<head>
  <meta charset="UTF-8">
  <title>{{.Title}}</title>
  <link rel="stylesheet" href="../css/list.css" />
</head>
<body>

<h2 id="{{.Id}}">{{.Title}}</h2>
<span class="origin"><a href="http://bbs.sanww.com/read.php?tid-{{.Id}}.html">原文链接</a></span>
<span id="page" val={{.Page}}>分页数: <strong>{{.Page}}</strong></span>
<p>

<div id="posts">
  <input class="search" placeholder="搜索题名或作者" />
  <button class="sort" data-sort="title">
    按名字排序
  </button>

  <p>说明：搜索可以输入搜索题名或作者的任意部分内容，下面列表将即时返回过滤结果。</p>

  <ul class="list">
    {{range .Follows}}<li>
      <span class="author" id="{{.AuthorId}}">作者：<a href="../users/{{.AuthorId}}.html"><strong>{{.Author}}</strong></a></span>
      <span class="post_time" val="{{.PostTime}}">发表时间：<strong>{{.PostTime}}</strong></span>
      <p class="text">{{.Text}}</p>
    </li>{{end}}
  </ul>

</div>
<script src="../js/jquery.min.js"></script>
<script src="../js/list.js"></script>

<script>

var options = {
  valueNames: [ 'title', 'author' ]
};

var userList = new List('posts', options);

</script>
</body>
</html>`
	tt, _ := template.New("webpage").Parse(tpl)
	file_url := "posts/" + strconv.Itoa(t.Id) + ".html"
	log.Printf("%-10s%s\n", "[Write]", file_url)
	fout, _ := os.Create(file_url)
	tt.Execute(fout, t)
}

func WriteTHtml2(t *T) {
	const tpl = `
<!DOCTYPE html>
<html lang="zh_CN">
<head>
  <meta charset="UTF-8">
  <title>{{.Title}}</title>
  <link rel="stylesheet" href="../css/list.css" />
</head>
<body>

<h2 id="{{.Id}}">{{.Title}}</h2>
<span class="origin"><a href="http://bbs.sanww.com/read.php?tid-{{.Id}}.html">原文链接</a></span>
<span id="page" val={{.Page}}>分页数: <strong>{{.Page}}</strong></span>
<p>

<div id="pages">
  <ul class="list">
  </ul>
</div>
<script src="../js/jquery.min.js"></script>
<script src="../js/list.js"></script>

<script>

for (var i = 1; i <= {{.Page}}; i++) {
    var url = "{{.Id}}_" + i.toString()+ ".html";
    var li = '<li><p><a href="'+url+'">第'+i.toString()+'页</a></p></li>';
    $("#pages ul.list").append(li);
}

var options = {
  valueNames: [ 'title', 'author' ]
};

var userList = new List('posts', options);

</script>
</body>
</html>`
	tt, _ := template.New("webpage").Parse(tpl)
	file_url := "posts/" + strconv.Itoa(t.Id) + ".html"
	log.Printf("%-10s%s\n", "[Write]", file_url)
	fout, _ := os.Create(file_url)
	tt.Execute(fout, t)
}

func WriteFHtml(f *F) {
	const tpl = `
<!DOCTYPE html>
<html lang="zh_CN">
<head>
  <meta charset="UTF-8">
  <title>{{.Title}}</title>
  <link rel="stylesheet" href="../css/list.css" />
</head>
<body>

<h2 id="{{.Id}}">{{.Title}}</h2>
<span id="title_count" val={{.TitleCount}}>主题: <strong>{{.TitleCount}}</strong></span>
<span id="full_count" val={{.FullCount}}>帖数: <strong>{{.FullCount}}</strong></span>
<span id="page" val={{.Page}}>分页数: <strong>{{.Page}}</strong></span>
<p>

<div id="posts">
  <input class="search" placeholder="搜索题名或作者" />
  <button class="sort" data-sort="title">
    按名字排序
  </button>

  <p>说明：搜索可以输入搜索题名或作者的任意部分内容，下面列表将即时返回过滤结果。</p>

  <ul class="list">
    {{range .Records}}<li>
      <h3 class="title" id={{.Id}}><a href="../posts/{{.Id}}.html">{{.Title}}</a></h3>
      <span class="author" id="{{.AuthorId}}">作者：<a href="../users/{{.AuthorId}}.html"><strong>{{.Author}}</strong></a></span>
      <span class="post_time" val="{{.PostTime}}">发表时间：<strong>{{.PostTime}}</strong></span>
      <span class="reply_count" val={{.ReplyCount}}>回帖：<strong>{{.ReplyCount}}</strong></span>
      <span class="view_count" val={{.ViewCount}}>浏览：<strong>{{.ViewCount}}</strong></span>
      <span class="last_reply" val="{{.LastReply}}">最后回复：<strong>{{.LastReply}}</strong></span>
      <span class="last_reply_time" val="{{.LastReplyTime}}">最后回复时间：<strong>{{.LastReplyTime}}</strong></span>
    </li>{{end}}
  </ul>

</div>
<script src="../js/jquery.min.js"></script>
<script src="../js/list.js"></script>

<script>

var options = {
  valueNames: [ 'title', 'author' ]
};

var userList = new List('posts', options);

</script>
</body>
</html>`
	tt, _ := template.New("webpage").Parse(tpl)
	file_url := "lists/" + strconv.Itoa(f.Id) + ".html"
	log.Printf("%-10s%s\n", "[Write]", file_url)
	fout, _ := os.Create(file_url)
	tt.Execute(fout, f)
}

func WriteFJson(f *F) {
	// Write to JSon
	file_url := "lists/" + strconv.Itoa(f.Id) + ".json"
	file, _ := os.Create(file_url)
	defer file.Close()
	log.Printf("%-10s%s\n", "[Write]", file_url)
	// b, _ := json.Marshal(f)
	b, _ := json.MarshalIndent(f, "", "    ")
	file.Write(b)
}

func WriteTJson(t *T) {
	// Write to JSon
	file_url := "posts/" + strconv.Itoa(t.Id) + ".json"
	file, _ := os.Create(file_url)
	defer file.Close()
	log.Printf("%-10s%s\n", "[Write]", file_url)
	// b, _ := json.Marshal(t)
	b, _ := json.MarshalIndent(t, "", "    ")
	file.Write(b)
}

// Read tid.json
func ReadTJson(tid int) (t *T) {
	t = new(T)
	file_url := "posts/" + strconv.Itoa(tid) + ".json"
	raw, _ := ioutil.ReadFile(file_url)
	log.Printf("%-10s%s\n", "[Read]", file_url)
	json.Unmarshal(raw, t)
	return
}

// Read fid.json
func ReadFJson(fid int) (f *F) {
	f = new(F)
	file_url := "lists/" + strconv.Itoa(fid) + ".json"
	raw, _ := ioutil.ReadFile(file_url)
	log.Printf("%-10s%s\n", "[Read]", file_url)
	json.Unmarshal(raw, f)
	return
}

func ProcessF(fids []int) {
	for _, fid := range fids {
		// Write out json file.
		// If already exist, ignore it
		file_url := "lists/" + strconv.Itoa(fid) + ".json"
		var f *F
		if _, err := os.Stat(file_url); err != nil {
			f, err = GetF(fid)
			if err != nil {
				log.Printf("%-10s%s%d\n", "[Error]", "GetF() fid=", fid)
				continue
			}
			WriteFJson(f)
		} else {
			log.Printf("%-10s%s\n", "[Ignore]", file_url)
			// Read data from json locally
			f = ReadFJson(fid)
		}
		WriteFHtml(f)
	}
}

func downloadFullPage(in_url, out_url string) (err error) {
	// wget <in_url> -p -O <out_url>
	cmd := "wget"
	args := []string{in_url, "-p", "-O", "out_url"}
	err = exec.Command(cmd, args...).Run()
	return
}

func ProcessT(fids []int) {
	for _, fid := range fids {
		f := ReadFJson(fid)

		for _, post := range f.Records {
			// Write out json file.
			// If already exist, ignore it
			file_url := "posts/" + strconv.Itoa(post.Id) + ".json"

			// If <tid>.json not exist
			var t *T
			if _, err := os.Stat(file_url); err != nil {
				t, err = GetT(post.Id)
				if err != nil {
					log.Printf("%-10s%s%d\n", "[Error]", "GetT() tid=", post.Id)
					continue
				}
				// Write json
				WriteTJson(t)
			} else {
				log.Printf("%-10s%s\n", "[Ignore]", file_url)
				// Read data from json locally
				t = ReadTJson(post.Id)
			}

			// Give up this solution, 'wget' instead !!!
			// WriteTHtml(t)
			//
			// Write HTML page for indexing at posts/<tid>.html
			WriteTHtml2(t)

			// Get HTML full page with `wget` if not exist yet
			var in_url, out_url string
			for page := 1; page <= t.Page; page++ {
				if page == 1 {
					in_url = fmt.Sprintf(URL_T, post.Id)
				} else {
					in_url = fmt.Sprintf(URL_T_PAGE, post.Id, page)
				}
				out_url = "posts/" + strconv.Itoa(post.Id) + "_" + strconv.Itoa(page) + ".html"
				fmt.Println(in_url, out_url)

				// if out_url not exist, download
				if _, err := os.Stat(out_url); err != nil {
					// downloadFullPage(in_url, out_url)
				}

			}
		}
	}
}

func main() {
	// Global initialize the decoder, used in everywhere
	ds = mahonia.NewDecoder("gbk").ConvertString

	ProcessF(fids)
	ProcessT(fids)

}
