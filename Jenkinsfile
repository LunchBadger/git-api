pipeline {
  agent {
    dockerfile {
      filename 'Dockerfile'
    }

  }
  stages {
    stage('error') {
      steps {
        git(url: 'git@github.com:LunchBadger/git-api.git', branch: 'master')
      }
    }
  }
}