package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type custom struct {
	ctx     context.Context
	baseURL string
	limit   *int
	query   *string
	path    string
	links   []string
	mu      sync.Mutex
}

func (c *custom) parseDoc(doc *goquery.Document, tag *[2]string, oneFlag bool) string {
	var link string

	sel := doc.Find(tag[0])
	if !oneFlag {
		for i := range sel.Nodes {
			if i == *c.limit {
				break
			}
			single := sel.Eq(i)
			link, _ = single.Attr(tag[1])
			c.links = append(c.links, link)
		}
	} else {
		single := sel.Eq(0)
		link, _ = single.Attr(tag[1])
		return link
	}
	return ""
}

func (c *custom) createFolder() error {

	_, err := os.Stat(c.path)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(c.path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *custom) saveUploadedFile(resp *http.Response, filename string) error {
	defer resp.Body.Close()

	c.mu.Lock()
	out, err := os.Create(filepath.Join(c.path, filename))
	if err != nil {
		return err
	}

	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	c.mu.Unlock()
	return err
}

func (c *custom) startDownload(url string, g *errgroup.Group) error {
	resp, err := getResponse(&url)
	if err != nil {
		return err
	}

	doc, err := getDoc(resp)
	if err != nil {
		return err
	}

	tag := [2]string{"aside .detail__actions .detail__download .selection-download-wrapper button", "data-href"}
	link := c.parseDoc(doc, &tag, true)

	if link != "" {
		g.Go(func() error {

			r, err := getResponse(&link)
			if err != nil {
				return err
			}
			defer r.Body.Close()

			fl := fmt.Sprintf("%s.jpg", filepath.Base(link))
			if err := c.saveUploadedFile(r, fl); err != nil {
				return err
			}

			return nil
		})
	}
	return nil
}

func (c *custom) start() error {

	c.query = flag.String("q", "cat", "Query name")       //get args
	c.limit = flag.Int("l", 3, "limit to download photo") //get args

	//с url скачивается не более 3 фото в день
	c.baseURL = fmt.Sprintf("https://ru.freepik.com/search?format=search&query=%s&type=photo", *c.query)
	c.links = make([]string, 0, *c.limit)
	c.path = filepath.Join("upload", *c.query)

	if err := c.createFolder(); err != nil {
		return err
	}

	flag.Parse()

	return nil
}

func getResponse(url *string) (*http.Response, error) {
	resp, err := http.Get(*url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch %v %v", resp.StatusCode, resp.Status)
	}
	return resp, nil
}

func getDoc(resp *http.Response) (*goquery.Document, error) {
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func main() {
	fatal := func(err error) {
		if err != nil {
			log.Fatal().Err(err).Send()
		}
	}

	c := custom{}
	err := c.start()
	fatal(err)

	resp, err := getResponse(&c.baseURL)
	fatal(err)

	doc, err := getDoc(resp)
	fatal(err)

	tag := [2]string{".list-content .showcase .showcase__item .showcase__content a", "href"}
	_ = c.parseDoc(doc, &tag, false)

	g, ctx := errgroup.WithContext(context.Background())
	c.ctx = ctx

	ch := make(chan string, len(c.links))
	for i := range c.links {
		ch <- c.links[i]
	}
	close(ch)

	for i := 0; i < 2; i++ {
		g.Go(func() error {
			for url := range ch {
				select {
				case <-ctx.Done():
					return nil
				default:
				}

				if err := c.startDownload(url, g); err != nil {
					return err
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		log.Info().Err(err).Send()
	}
}
