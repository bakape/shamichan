package templates

// type noscriptVars struct {
// 	Title, DefaultCSS string
// 	Threads           template.HTML
// 	Boards            []string
// }

// type threadVars struct {
// 	Notice, Title string
// 	Thread        *types.Thread
// }

// // Common part of both thread and board noscript pages
// func renderNoscriptIndex(data []byte, title string) ([]byte, error) {
// 	w := new(bytes.Buffer)
// 	boards := config.GetBoards()
// 	sort.Strings(boards)

// 	err := tmpl["noscript"].Execute(w, noscriptVars{
// 		Threads:    template.HTML(data),
// 		Boards:     append([]string{"all"}, boards...),
// 		DefaultCSS: config.Get().DefaultCSS,
// 		Title:      title,
// 	})
// 	return w.Bytes(), err
// }

// // Thread renders thread page HTML for noscript browsers
// func Thread(t *types.Thread) ([]byte, error) {
// 	w := new(bytes.Buffer)
// 	conf := config.GetBoardConfigs(t.Board)
// 	title := fmt.Sprintf("/%s/ - %s (#%d)", t.Board, t.Subject, t.ID)

// 	v := threadVars{
// 		Notice: conf.Notice,
// 		Title:  title,
// 		Thread: t,
// 	}

// 	err := tmpl["thread"].Execute(w, v)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return renderNoscriptIndex(w.Bytes(), title)
// }
