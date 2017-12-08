services: docker
script:
  - docker build -t meguca_test .
  - docker run -t --entrypoint scripts/docker_test.sh meguca_test
