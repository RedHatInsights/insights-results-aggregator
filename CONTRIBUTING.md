# Contributing

## General code rules

For docstrings we're following [Effective Go - Commentary](https://golang.org/doc/effective_go.html#commentary)

All methods meant to be used by end-user *must be* documented - **no exceptions!**

Other code quality metrics are measured as well via [Go Report Card](https://goreportcard.com/report/github.com/RedHatInsights/insights-results-aggregator)


### Naming conventions

Please try to follow [Style guideline for Go packages](https://rakyll.org/style-packages/) and [Package names](https://blog.golang.org/package-names)


## Submitting a pull request

Before you submit your pull request consider the following guidelines:

* Fork the repository and clone your fork
  * open the following URL in your browser: https://github.com/RedHatInsights/insights-results-aggregator
  * click on the 'Fork' button (near the top right corner)
  * select the account for fork
  * open your forked repository in browser: https://github.com/YOUR_NAME/insights-results-aggregator
  * click on the 'Clone or download' button to get a command that can be used to clone the repository

* Make your changes in a new git branch:

     ```shell
     git checkout -b bug/my-fix-branch master
     ```

* Create your patch, **ideally including appropriate test cases**
* Include documentation that either describe a change to a behavior or the changed capability to an end user
* Commit your changes using **a descriptive commit message**. If you are fixing an issue please include something like 'this closes issue #xyz'
    * Please make sure your tests pass!
    * Currently we use Travis CI for our automated testing.

* Push your branch to GitHub:

    ```shell
    git push origin bug/my-fix-branch
    ```

* When opening a pull request, select the `master` branch as a base.
* Mark your pull request with **[WIP]** (Work In Progress) to get feedback but prevent merging (e.g. [WIP] Update CONTRIBUTING.md)
* If we suggest changes then:
  * Make the required updates
  * Push changes to git (this will update your Pull Request):
    * You can add new commit
    * Or rebase your branch and force push to your GitHub repository:

    ```shell
    git rebase -i master
    git push -f origin bug/my-fix-branch
    ```

That's it! Thank you for your contribution!
