param(
    [string]$Owner = "flaviokalleu",
    [string]$Repo = "germanio",
    [string]$Version = "v0.2.0",
    [switch]$CreateRelease
)

$ErrorActionPreference = "Stop"

$branch = (git branch --show-current).Trim()
if (-not $branch) {
    $branch = "master"
}

$description = "Germanio is a bilingual declarative programming language for building full-stack web applications from .ge files."
$homepage = "https://github.com/$Owner/$Repo/tree/$branch/docs"
$topics = @(
    "germanio",
    "programming-language",
    "declarative-language",
    "dsl",
    "compiler",
    "interpreter",
    "golang",
    "full-stack"
)

$releaseNotes = @"
Germanio $Version

Highlights
- Bilingual declarative programming language with .ge source files
- Compiler pipeline with lexer, parser, and AST in Go
- Embedded runtime with REST API, dashboard, WebSocket, auth, and database support
- Example programs in examples/ and larger apps in demo/

Repository metadata
- About description configured
- Homepage set to docs/
- Topics configured for discoverability

Suggested next step
- Commit and push the current branch before publishing this release so the tag matches the latest repository structure.
"@

Write-Host "Checking GitHub CLI authentication..."
gh auth status | Out-Null

$repoRef = "$Owner/$Repo"

Write-Host "Updating repository metadata for $repoRef..."
gh repo edit $repoRef --description $description --homepage $homepage | Out-Null

foreach ($topic in $topics) {
    gh repo edit $repoRef --add-topic $topic | Out-Null
}

Write-Host "Repository metadata updated."

if ($CreateRelease) {
    $hasUncommitted = git status --porcelain
    if ($hasUncommitted) {
        throw "There are uncommitted changes in the working tree. Commit and push before creating a release."
    }

    $existingTag = git tag --list $Version
    if (-not $existingTag) {
        Write-Host "Creating git tag $Version..."
        git tag $Version
    }

    Write-Host "Pushing tag $Version..."
    git push origin $Version

    Write-Host "Creating GitHub release $Version..."
    gh release create $Version --repo $repoRef --title "Germanio $Version" --notes $releaseNotes
    Write-Host "Release created."
}
else {
    Write-Host "Skipping release creation. Run with -CreateRelease after committing and pushing changes."
}