# recipes for manipulating 'rejection' files created when there are mismatches
# between rendered output and expected images.

list_rejects:
	@find . -name testdata -type d | while read testdatadir ; do \
		find "$$testdatadir" -name '*.rej.*' ; \
	done

# opens expected and rejected files in 'feh'
view_rejects:
	@find . -name testdata -type d | while read testdatadir ; do \
		find "$$testdatadir" -name '*.rej.*' | while read rejfile ; do \
			echo -e >&2 "$${rejfile/.rej}\n$$rejfile" ; \
			echo "$${rejfile/.rej}" "$$rejfile" ; \
		done ; \
	done | xargs -r feh

clean_rejects:
	find . -name testdata -type d | while read testdatadir ; do \
		find "$$testdatadir" -name '*.rej.*' -exec rm "{}" + ; \
	done

promote_rejects:
	@find . -name testdata -type d | while read testdatadir ; do \
		find "$$testdatadir" -name '*.rej.*' | while read rejfile ; do \
			echo mv "$$rejfile" "$${rejfile/.rej}" ; \
			mv "$$rejfile" "$${rejfile/.rej}" ; \
		done \
	done

.PHONY: list_rejects view_rejects clean_rejects promote_rejects
