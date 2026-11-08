.PHONY: setup test

setup:
	git clone https://github.com/codecrafters-io/course-sdk course_sdk

test:
	WAF_COURSE_DIR="$(CURDIR)" $(MAKE) -C ../firewall-tester test
	python3 -m unittest discover -s examples/waf-lab -p 'test_*.py'
