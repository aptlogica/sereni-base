pipeline {
  agent any

  stages {
    stage('Checkout Code') {
      steps {
        checkout scm
      }
    }

    stage('Test With Coverage') {
      steps {
        sh 'mkdir -p coverage'
        sh 'go test -v -race -coverprofile=coverage.out -covermode=atomic -coverpkg=./... ./...'
      }
    }

    stage('SonarQube Analysis') {
        when {
        anyOf {
          branch 'develop'
          branch 'main'
          branch 'release/*'
          branch 'master'
        }
      }
      steps {
        script {
          // Get path to the installed Sonar Scanner tool
          def scannerHome = tool 'SonarScanner'
          
          withSonarQubeEnv('aptl-sonar') {
            // Run the scanner binary
            sh "${scannerHome}/bin/sonar-scanner"
          }
        }
      }
    }

    stage('Quality Gate') {
      when {
        anyOf {
          branch 'develop'
          branch 'main'
          branch 'release/*'
          branch 'master'
        }
      }
      steps {
        timeout(time: 10, unit: 'MINUTES') {
          waitForQualityGate abortPipeline: true
        }
      }
    }
  }
}
