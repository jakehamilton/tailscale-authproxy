{ lib, mkShell, go_1_19, gopls, ... }:

mkShell {
  buildInputs = [ go_1_19 gopls ];
}
