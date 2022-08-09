* Bugs 
- [x] Fix bug with guessit and filename?
- [x] Running more than 1 test
- [x] Move the database away from config directory
- [ ] Can we fix "stranger things s04e09" not being 50 min but 150min?

* 0.1 movix.go 
  - [ ] Delete (and suggest files for deletion)
  -- [ ] Can we unify suggestions, so that we get suggested to delete all episodes 
  in a show, we should just return the show instead of all entries.
  -- [ ] With a unified suggestion, can we have a unified delete?
  - [ ] entry id shouldn't be the foreign id, but local 

  - [x] Write a basic test suit
  - [-] Support searches
  - [ ] Write transmission script
  -- [ ] Write a test script that sets all variables so you can test it without a callback
  -- [ ] Test it
  - [x] Move support
  - [x] Deleting files should mark them as deleted in the db
  - [x] Cleaning up the code.
  - [x] Split the code into different files
  - [x] fuzzy scripts
  -- [x] fzf
  -- [x] dmenu family
  - [x] Support more format
  -- [x] Support movies

  - [x] Add logging

  - [ ] Add a filter rule

  - [ ] Testing code 
  -- [x] Get testing code working
  -- [ ] Make it easier integrate test code
  -- [ ] Make it easier to test parts

- [ ] Get move to work

* Packaging
- [x] Readme
-- [x] Usage
-- [x] Install
- [x] Package it so it's easier for others to install

* 0.2
- [ ] change skipUntil etc to range functions that takes a function and applies it to every 
value in the range
  - [ ] A way to suggest moving of the content. That way we don't need to stop the torrent and we don't need "delete" things.
  -- We just ask for a path, lets transmission move to that path and add that path
  (- [ ] Write a guessit in go (for a speedup))
  - [ ] Sync
  - [ ] Transfer?
  - [ ] Undo?
  - [ ] Do multiple workers for add (and maybe for other stuff)
  - [ ] A run-or-bring mode(?, this is mainly the wm?)
  - [ ] Tag support (like: edu, comedy etc)
  -- [ ] Tag based searching

  -- [ ] Support vods
  -- [ ] Audiobooks
