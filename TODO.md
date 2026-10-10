# Todos

* [x] **Fix keys printSlice**
  - Was creating empty deque then printing it.
* [ ] **Add type switch to try to auto cast cli strs to json compat types**
  - It'd be dope if when someone ran `push deque 3` it knew that was an int instead of just making everything from the cli a str
  - Maybe also allow users to still say str 3 with like `'"3"' or something somehow...
  - default to str if can't find json compatible type.
* [ ] **Generic cleanup and auditing.**
  - There's still misc bugs I gotta hunt down, testing will help.
  - Also need to tweak helpers to be more _idiomatic_.
* [ ] **Add more debug prints for debug mode**
  - Right now prints dequeMap via `spew` before modifying (after reading json)
* [x] **Rewrite to use Deque Struct + Struct methods**
  - Can do arbitrary typing this way
* [ ] **Write test code**
* [ ] **Write github actions yaml**
  - Dev Branch 
    - Linter
    - Run tests
    - Run coverage
  - Merge to main
    - Build binaries
      - win, lin, mac
      - x86, arm64
    - GoReleaser to build debs
    - Push deb to github pages
* [ ] **Setup Github Pages w/ custom domain**
  - Thinking https://pkgs.thefamu.net
* [ ] **...**
* [ ] **Profit!**
  - Jk, its foss
