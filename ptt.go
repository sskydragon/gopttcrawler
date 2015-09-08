package gopttcrawler

import (
	"errors"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const (
	BASE_URL = "https://www.ptt.cc/bbs/"
)

type Article struct {
	ID       string //Article ID
	Board    string //Board name
	Title    string
	Content  string
	Author   string //Author ID
	DateTime string
	Nrec     int //推文數(推-噓)
}

type ArticleList struct {
	Articles     []*Article //Articles
	Board        string     //Board
	PreviousPage int        //Previous page id
	NextPage     int        //Next page id
}

func newDocument(url string) (*goquery.Document, error) {
	// Load the URL
	res, e := http.Get(url)
	if e != nil {
		return nil, e
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New(res.Status)
	}

	return goquery.NewDocumentFromResponse(res)
}

func getPage(prefix, text string) int {
	re := regexp.MustCompile(prefix + "/index(\\d+).html$")
	matched := re.FindStringSubmatch(text)

	if len(matched) > 1 {
		ret, _ := strconv.Atoi(matched[1])
		return ret
	}

	return 0
}

func GetArticles(board string, page int) (*ArticleList, error) {
	index := "/index.html"
	if page != 0 {
		index = "/index" + strconv.Itoa(page) + ".html"
	}

	url := BASE_URL + board + index
	doc, err := newDocument(url)

	if err != nil {
		return nil, err
	}

	articleList := &ArticleList{PreviousPage: 0, NextPage: 0, Board: board}

	prevPageSel := doc.Find(".action-bar").Find("a:contains('上頁')")
	if len(prevPageSel.Nodes) > 0 {
		href, _ := prevPageSel.Attr("href")
		articleList.PreviousPage = getPage("/bbs/"+board, href)
	}
	nextPageSel := doc.Find(".action-bar").Find("a:contains('下頁')")
	if len(nextPageSel.Nodes) > 0 {
		href, _ := nextPageSel.Attr("href")
		articleList.NextPage = getPage("/bbs/"+board, href)
	}

	articles := make([]*Article, 0)
	doc.Find(".r-ent").Each(func(i int, s *goquery.Selection) {
		article := &Article{Board: board}
		//Nrec
		nrecSel := s.Find(".nrec")
		if len(nrecSel.Nodes) > 0 {
			article.Nrec, _ = strconv.Atoi(nrecSel.Text())
		}
		//DateTime
		DateTimeSel := s.Find(".DateTime")
		if len(DateTimeSel.Nodes) > 0 {
			article.DateTime = DateTimeSel.Text()
		}
		//Author
		authorSel := s.Find(".author")
		if len(authorSel.Nodes) > 0 {
			article.Author = authorSel.Text()
		}
		//Title
		linkSel := s.Find(".title > a")
		if linkSel.Size() != 0 {
			href, existed := linkSel.Attr("href")
			if existed {
				re := regexp.MustCompile("/bbs/" + board + "/(.*).html$")
				matchedID := re.FindStringSubmatch(href)
				if matchedID != nil && len(matchedID) > 1 {
					article.ID = matchedID[1]
					article.Title = strings.TrimSpace(linkSel.Text())
					articles = append(articles, article)
				}
			}
		}
	})

	articleList.Articles = articles

	return articleList, nil
}

func LoadArticles(board, id string) (*Article, error) {
	url := BASE_URL + board + "/" + id + ".html"

	doc, err := newDocument(url)

	if err != nil {
		return nil, err
	}

	article := &Article{ID: id, Board: board}

	//Get title
	article.Title = strings.TrimSpace(doc.Find("title").Text())
	//Get Content
	meta := doc.Find(".article-metaline")
	meta.Find(".article-meta-value").Each(func(i int, s *goquery.Selection) {
		switch i {
		case 0: //Author
			name := s.Text()
			re := regexp.MustCompile("^(.*)\\s+\\(.*\\)")
			matched := re.FindStringSubmatch(name)

			if matched != nil && len(matched) > 1 {
				name = matched[1]
			}
			article.Author = name
		case 2: //Time
			article.DateTime = strings.TrimSpace(s.Text())
		}
	})
	
	meta.Remove() //Remove header

	//Remove board name
	metaRight := doc.Find(".article-metaline-right")
	metaRight.Remove()

	push := doc.Find(".push")
	//Count push
	pushCnt := doc.Find(".push:contains('推')").Size()
	booCnt := doc.Find(".push:contains('噓')").Size()
	article.Nrec = pushCnt - booCnt
	push.Remove()

	sel := doc.Find("#main-content")
	article.Content, _ = sel.Html()

	return article, nil
}

func (aList *ArticleList) GetFromPreviousPage() (*ArticleList, error) {
	return GetArticles(aList.Board, aList.PreviousPage)
}

func (aList *ArticleList) GetFromNextPage() (*ArticleList, error) {
	return GetArticles(aList.Board, aList.NextPage)
}

func (a *Article) Load() *Article {
	newA, err := LoadArticles(a.Board, a.ID)
	if err == nil {
		*a = *newA
	}
	return a
}

func (a *Article) GetImageUrls() ([]string, error) {
	url := BASE_URL + a.Board + "/" + a.ID + ".html"

	doc, err := newDocument(url)

	if err != nil {
		return nil, err
	}

	result := make([]string, 0)
	imgs := doc.Find("#main-content").Find("img")
	imgs.Each(func(i int, s *goquery.Selection) {
		src := s.AttrOr("src", "")
		if src != "" {
			result = append(result, src)
		}
	})
	return result, nil
}

func (a *Article) GetLinks() ([]string, error) {
	url := BASE_URL + a.Board + "/" + a.ID + ".html"

	doc, err := newDocument(url)

	if err != nil {
		return nil, err
	}

	result := make([]string, 0)
	links := doc.Find("#main-content").Find("a")
	links.Each(func(i int, s *goquery.Selection) {
		src := s.AttrOr("href", "")
		if src != "" {
			result = append(result, src)
		}
	})
	return result, nil
}
