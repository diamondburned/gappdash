{ pkgs ? import ./pkgs.nix {} }:

let src = import ./src.nix;

in pkgs.mkShell {
	buildInputs = with pkgs; [
		gnome.gtk3
		glib
		gtk-layer-shell
		gdk-pixbuf
		gobjectIntrospection
	];

	nativeBuildInputs = with pkgs; [
		go
		pkgconfig
	];

	CGO_ENABLED = 1;
}
