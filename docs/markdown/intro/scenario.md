# Scenario

Here's a complex scenario that shows some of the possibilities of using Sugarkube:

Imagine you're the developer for a web site. It runs on Kubernetes in production but you develop on Minikube locally. You want to optimise it to cache data in Memcached. While developing locally you want to develop against a Memcached instance in your Minikube cluster, but on AWS you'll use ElastiCache. You've already created a kapp for the web site. It's deployed as a Helm chart, and the current live release is v2.4.0. Your web site requires an S3 bucket and uses RDS in production but the Terraform code to create these is already part of the kapp.

You could approach this problem in two ways:

1. Create a kapp for Memcached and install it before your website kapp, or
1. Conditionally create a Memcached deployment or spin up an ElastiCache cluster depending on whether your web site is being run against a local target or AWS account.

The first approach would be better if you plan to have multiple web sites use ElastiCache in future because it'd be a kapp for a shared service. Alternatively you might also want to go with the first approach because you'll be creating a reusable kapp for future use. The second approach mixes concerns so in this situation isn't as neat. However, it does illustrate that you have a choice in how to solve this problem.

After some discussion you decide to go with the first approach so you'll have a reusable kapp for the future.

You create a kapp for Memcached, which contains a Helm chart that installs Memcached, and a directory `terraform_aws` that creates an ElastiCache cluster. You customise the Makefile so it only installs the Helm chart when running locally. The Terraform code will only be executed when targetting AWS because of the `_aws` suffix. This functionality is provided 
by our default Makefiles so you get this with no effort if you use them.

After configuring Sugarkube to install your Memcached kapp, you use Sugarkube to create a Minikube cluster. You've already got kapps for Tiller and nginx-ingress and they're automatically installed before installing the kapps for Memcached and your web site. Your web site kapp also installs some fixture data when running in non-live environments.

A short while later you have a local Minikube cluster configured and ready to go. You create a feature branch for your web site and make the necessary changes to make it use Memcached (your web site is running in the Minikube cluster by mounting a local shared directory). Once you're happy, you commit the changes to your website and push your git repo. This triggers your CI/CD system to create a new docker image tagged with your feature branch name. The hostname for your Memcached cluster is set as an environment variable at run-time. The last thing to do is to create a feature branch for your web site's kapp to make it use the new docker image. There's no need to commit or push your changes for this branch yet.

At this point, you have the following:

* A Memcached kapp that will either install a Memcached pod into a K8s cluster when running locally, or use Terraform to create an ElastiCache cluster when targetting AWS.
* A new, tagged docker image for your web site that's been updated to pull data from Memcached. The host name for Memcached is configured through environment variables. You'll promote this exact image through multiple environments on it's route into the live environment
* A kapp for your web site that's been updated to use the new docker image created above.
  
The next step is to test your work on your dev AWS cluster. Because Sugarkube is just a binary, you can easily run it locally even targetting one of your AWS clusters without messing around with Jenkins. You update the Sugarkube manifest so it'll deploy from your feature branch into your dev cluster and manually run it so it installs your kapps. Because Sugarkube passes different environment variables to your Makefile, instead of installing Memcached as a pod this time, it runs the Terraform code in `terraform_aws` to create an ElastiCache cluster. The RDS database and S3 bucket already exist so there are no changes there.

You test it and everything looks good, so you commit the changes to your web site kapp repo, then open a PR to merge it into master which makes your CI pipeline tag your kapps repo at this version. It does this by inspecting which directories were modified in the previous commit and tagging them according to the tag in the kapp's `sugarkube.yaml` file.

Now, your kapp repo has been updated and tagged with this new release. Next you create a feature branch for the repo containing your Sugarkube configs and update it to release your new kapp into staging. You commit and merge this and your CI system runs Sugarkube again, targetting your staging AWS cluster.

Again, you test, you're happy, so you update the repo containing your Sugarkube configs to release into prod. This triggers Sugarkube to release into production. If you're really advanced your CI pipeline also pushes the docker image through various container repos as it progresses.

**Note**: The above approach requires you to explicitly control which environment you're releasing into. This lets you support multiple live environments which is sometimes a requirement. For simpler setups, you could decide not to explicitly define the versions of kapps to release into staging, but instead modify your CI pipeline so that before releasing into prod it automatically releases into staging, runs tests, then releases to prod if everything is OK.

The trade-offs Sugarkube has chosen give you ultimate control over how much automation you want vs how much explicit intervention should be required to release your changes.
