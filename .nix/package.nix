{
	internalPkgs ? import ./pkgs.nix {}, # only for overriding
	pkgs ? internalPkgs,
	lib ? pkgs.lib,
	src ? null,
	doCheck ? false,
	vendorSha256 ? lib.fakeSha256,
}:

let desktopFile = pkgs.makeDesktopItem {
    desktopName = "Application Launcher";
	name = "GAppDash";
	exec = "gappdash";
	icon = "start-here";
};

in internalPkgs.buildGoModule {
	inherit src doCheck vendorSha256;

	pname = "gappdash";
	version = "0.0.1-tip";

	buildInputs = with pkgs; [
		gnome.gtk3
		gtk-layer-shell
		gdk-pixbuf
		gobjectIntrospection
		librsvg

		glib
		gnome3.defaultIconTheme
	];

	nativeBuildInputs = with pkgs; [
		pkgconfig

		wrapGAppsHook
		glib
		gnome3.defaultIconTheme
	];

	preFixup = ''
		mkdir -p $out/share/icons/hicolor/256x256/apps/ $out/share/applications/
		cp "${desktopFile}"/share/applications/* $out/share/applications/
	'';
}
