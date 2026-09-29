"""Validate typed arguments without changing files, services or network state."""
import sys

project, count, enabled = sys.argv[1:]
assert project and project[0].islower()
assert all(character.islower() or character.isdigit() or character == '-' for character in project)
assert int(count) in range(1, 11)
assert enabled in ('true', 'false')
