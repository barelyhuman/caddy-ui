watch: 
	find . -name '*.go' -o -name '*.html' | entr -r go run .

w: watch
	