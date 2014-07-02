# ToDo List

[ ] Consolidate the DB modification into a more central location, rather than
    having it spread out across the entire codebase
[ ] Remove the race condition between loading a check to update and it being
    deleted (perhaps: keep the db.Update open during the update itself?)
    [ ] Ensure that an update that takes a long time doesn't block reads


# Maybe Later

[ ] Allow checks to be scripted using an embedded language:
    - https://github.com/zhemao/glisp/wiki/Language
    - https://github.com/robertkrimen/otto
    - https://github.com/stevedonovan/luar/ (not statically linked)
    - http://godoc.org/launchpad.net/twik


# Plans

CheckManager, initialized with DB & cron instance
  - Functions
    - `GetCheck *Check`
    - ModifyCheck
    - RunCheck
      - Should return whether anything changed
    - AddCheck
    - DeleteCheck
  - Automatically handles scheduling Cron runs
  - Shouldn't run deleted checks
  - Maybe: re-create cron instance if there are more than N deletions?
