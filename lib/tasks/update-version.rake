desc "Update version file"

task :update_version do
  current_version = File.read(".version").strip
  next_version = (Integer(current_version) + 1).to_s

  branch="update-version-#{next_version}"

  %x(git checkout -b "#{branch}")
  %x(git pull origin "#{branch}")

  File.write(".version", next_version)

  %x(git add ".version")
  %x(git remote set-url origin https://#{ENV["GOVUK_CI_GITHUB_API_TOKEN"]}@github.com/alphagov/govuk-synthetic-test-app.git)
  %x(git commit -m "Update version to #{next_version}")
  %x(git push --set-upstream origin "#{branch}")

  %x(git checkout update-version-main)
  %x(git pull origin update-version-main)
  %x(git merge --no-ff #{branch})
  %x(git push origin update-version-main)

  %x(git checkout update-version)
  %x(git reset --hard origin/update-version)
end
