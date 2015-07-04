PID      = /tmp/oddjobs-v2.pid
GO_FILES = $(wildcard *.go)
 
serve:
		@make restart 
		@fswatch -o . | xargs -n1 -I{}  make restart || make kill
			 
kill:
		@kill `cat $(PID)` || true
		 
stuff:
		@echo "actually do nothing"
		 
restart:
		@make kill
		# @make stuff
		@go run $(GO_FILES) & echo $$! > $(PID)
				 
#.PHONY: serve restart # kill stuff  let's go to reserve rules names

