#!/usr/bin/env bats

GOGITIX=$PWD/../gogitix

@test "gogitix with default options" {
  cd sample-project
  run $GOGITIX
  [ $status -eq 0 ]
}

@test "gogitix with lndir" {
  cd sample-project
  run $GOGITIX -lndir
  [ $status -eq 0 ]
}
