private:
	-mv .gitignore .gitignore_bak
	cp .gitignore_private .git/info/exclude
	git add .
	-git commit -m "$(msg)"
	git push private main
	-mv .gitignore_bak .gitignore

public:
	rm -f .git/info/exclude
	git add .
	-git commit -m "$(msg)"
	git push public main