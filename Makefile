.PHONY: compare
compare: diff clean

.PHONY: diff
diff: go-gen sonobuoy-gen
	@diff sono-gen.yaml go-gen.yaml
	

.PHONY: clean
clean:
	@rm -f go-gen.yaml
	@rm -f sono-gen.yaml

.PHONY: go-gen
go-gen:
	@go run main.go > go-gen.yaml

.PHONY: sonobuoy-gen
sonobuoy-gen:
	@sonobuoy gen -p=e2e --e2e-focus='\[Conformance\]' --e2e-skip='' --e2e-repo-config=conformance-image-config.yaml > sono-gen.yaml
