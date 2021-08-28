
# Sync

![Screenshot](https://github.com/kyprifog/sync/blob/master/images/screenshot.png)

This is a github sync tool.  Currently only works with master branch.  Its intended to keep your master branch in sync so you don't forget to pull before creating a feature branch.

This displays to you whether the branch is out of date or not.  Red == Remote has changes which you do not have locally. Yellow == Local changes not pushed up to remote.  Pressing the button will then sync it (pull down prioritizing `theirs`).  Typically for a master branch you are not making changes directly so this is ideal.

Sync is powered by ![tcell](https://github.com/gdamore/tcell)

## Usage

```
go build
```

```
./sync [REPOS_CONFIG_PATH]

```

Then place a repos.yaml file at `~/.repos.yaml` resembling:

```
repos:
  - name: <CHOSE A NAME>
    path: <PATH ON DISK>
    push: <true | false>
```

Push determines whether or not you want this tool to automatically push local changes when pressing sync (in some scenarios this may not be ideal since you want to tailor the commit message).

Run using `./todo`, quit by pressing `Esc + Esc`
