{
	src ? ./..,
	lib,
	pkgs,
	internalPkgs ? import ./pkgs.nix {}, # only for overriding
	vendorSha256 ? lib.fakeSha256,
}:

let desktopFile = pkgs.makeDesktopItem {
    desktopName = "Application Dash";
	name = "GAppDash";
	icon = "gappdash";
	exec = "gappdash";
};

in internalPkgs.buildGoModule {
	inherit src vendorSha256;

	pname = "gappdash";
	version = "0.0.1-tip";

	buildInputs = with internalPkgs; [
		gnome.gtk3
		glib
		gtk-layer-shell
		gdk-pixbuf
		gobjectIntrospection
	];

	nativeBuildInputs = with pkgs; [ pkgconfig ];

	preFixup = ''
		mkdir -p $out/share/icons/hicolor/256x256/apps/ $out/share/applications/
		# Install the desktop file
		cp "${desktopFile}"/share/applications/* $out/share/applications/
		# Install the icon
		cp "${../.github/logo-256.png}" $out/share/icons/hicolor/256x256/apps/gotktrix.png
	'';
}
