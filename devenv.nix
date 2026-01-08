{
  pkgs,
  lib,
  config,
  inputs,
  ...
}: {
  languages.go.enable = true;

  # https://devenv.sh/processes/
  # processes.dev.exec = "${lib.getExe pkgs.watchexec} -n -- ls -la";
  processes.web.exec = "go run ${config.git.root}/cmd/zoekt-webserver -index .zoekt";

  scripts = {
    zoekt.exec = "go run ${config.git.root}/cmd/zoekt $@";
    zoekt-git-index.exec = "nice -n 19 ionice -c 2 -n 7 go run ${config.git.root}/cmd/zoekt-git-index $@";
  };
  # https://devenv.sh/tasks/
  # tasks = {
  #   "myproj:setup".exec = "mytool build";
  #   "devenv:enterShell".after = [ "myproj:setup" ];
  # };

  # https://devenv.sh/git-hooks/
  # git-hooks.hooks.shellcheck.enable = true;

  # See full reference at https://devenv.sh/reference/options/
}
