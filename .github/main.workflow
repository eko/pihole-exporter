workflow "Build and Publish" {
  on = "push"
  resolves = ["Master", "Docker Push"]
}

# Filter for master branch
action "Master" {
  uses = "actions/bin/filter@master"
  args = "branch master"
}

action "Docker Build" {
  uses = "actions/docker/cli@fe7ed3ce992160973df86480b83a2f8ed581cd50"
  needs = ["Master"]
  args = "build -t pihole-exporter ."
}

action "Docker Registry" {
  uses = "actions/docker/login@fe7ed3ce992160973df86480b83a2f8ed581cd50"
  env = {
    DOCKER_REGISTRY_URL = "docker.pkg.github.com"
  }
  secrets = ["DOCKER_PASSWORD", "DOCKER_USERNAME"]
  needs = ["Docker Build"]
}

action "Docker Push" {
  uses = "actions/docker/cli@fe7ed3ce992160973df86480b83a2f8ed581cd50"
  args = "push pihole-exporter"
  needs = ["Docker Registry"]
}
