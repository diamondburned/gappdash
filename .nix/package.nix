{
	internalPkgs ? import ./pkgs.nix {}, # only for overriding
	pkgs ? internalPkgs,
	lib ? pkgs.lib,
	src ? ./..,
	doCheck ? false,
	vendorSha256 ? lib.fakeSha256,
}:

let desktopFile = pkgs.makeDesktopItem {
    desktopName = "Application Dash";
	name = "GAppDash";
	# icon = "gappdash";
	exec = "gappdash";
};

in internalPkgs.buildGoModule {
	inherit src doCheck vendorSha256;

	pname = "gappdash";
	version = "0.0.1-tip";

	buildInputs = with pkgs; [
		gnome.gtk3
		glib
		gtk-layer-shell
		gdk-pixbuf
		gobjectIntrospection
	];

	nativeBuildInputs = with pkgs; [
		pkgconfig
		wrapGAppsHook
	];

	preFixup = ''
		mkdir -p $out/share/icons/hicolor/256x256/apps/ $out/share/applications/
		cp "${desktopFile}"/share/applications/* $out/share/applications/
	'';
}
